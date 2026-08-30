# Vector approximate-neighbor profiles

Vectors have no useful database-independent bytewise total order. Schottky therefore stores the original vector as the exact value and emits derived `Float32` projection fields for B-tree candidate retrieval. Every approximate result must be reranked with the original vector.

This guide defines two best-effort profiles:

- trained PCA projections for stable datasets;
- training-free Gaussian projections for QALSH-style multi-index retrieval.

Neither profile promises a recall level for an unseen distribution. Measure Recall@K on held-out queries before activating a profile.

## Recommended storage model

The original vector is authoritative. Projection rows are rebuildable index data tied to an immutable profile.

| Value | Storage | Invariant |
| --- | --- | --- |
| original vector | application-owned Float32 blob or native vector value | dimension and element format are schema metadata |
| projection profile | canonical `ProjectionProfile.MarshalBinary` bytes | SHA-256 fingerprint identifies exact mean and axes |
| projection value | native binary32 value | used to merge lower and upper index cursors |
| projection key | Schottky `Float32` field | five bytes: presence tag plus sortable binary32 payload |
| profile identifier | schema-managed integer or digest | every item and projection row names one profile |

Normalize cosine-search vectors through `VectorL2`. Unit vectors make cosine, dot-product, and squared Euclidean ordering equivalent.

### Serialized projection profile

`MarshalBinary` emits this canonical big-endian layout:

| Byte range | Value |
| --- | --- |
| `0..3` | ASCII `SKVP` |
| `4` | `ProjectionProfileVersion` |
| `5` | projection method |
| `6` | vector normalization |
| `7` | bit 0 indicates a stored mean |
| `8..11` | unsigned dimension count |
| `12..15` | unsigned projection count |
| remaining | optional mean followed by row-major axes, all IEEE binary32 |

PCA profiles contain one dimension-length mean. Gaussian profiles contain no mean. Reject unknown versions or flags instead of guessing their meaning.

## Profile selection

| Requirement | Profile | Initial configuration | Limitation |
| --- | --- | --- | --- |
| no fitting or corpus scan | Gaussian multi-projection | 16 projections for write-sensitive partitions | measured 90% recall required reranking most of the corpus |
| stable corpus and lower read amplification | trained PCA | one projection | profile rebuild required after material distribution drift |
| read-heavy analytical partition without training | Gaussian collisions | 64–128 projections | large projection table and query fan-out |
| small filtered partition | exact scan | no projection profile | cost grows linearly with filtered row count |

PCA was the most selective measured B-tree prefilter. Gaussian projections remain the recommended **training-free storage format**, not the default query plan when training is allowed.

## Go profile lifecycle

### Training-free Gaussian profile

Generate the axes once, serialize the profile, and persist its fingerprint. Do not regenerate persisted keys from the seed.

```go
profile, err := schottky.NewGaussianProjectionProfile(
    schottky.GaussianProjectionOptions{
        Dimension:     1536,
        Projections:   16,
        Seed:          20260830,
        Normalization: schottky.VectorL2,
    },
)
if err != nil {
    return err
}

profileBytes, err := profile.MarshalBinary()
if err != nil {
    return err
}
profileID := profile.Fingerprint()
```

The generator uses independent Gaussian axes. The seed is an initialization convenience inside one implementation. The serialized Float32 axes are the portable contract.

### PCA training

Train only on the corpus side of an evaluation split. Queries used to measure recall must not participate in training.

```go
profile, err := schottky.TrainPCA(samples, schottky.PCAOptions{
    Projections:   1,
    Normalization: schottky.VectorL2,
    MaxIterations: 256,
    Tolerance:     1e-8,
})
if err != nil {
    return err
}
```

`TrainPCA` performs these steps:

1. validate a rectangular finite Float32 sample matrix;
2. apply the configured normalization;
3. calculate the sample mean;
4. center the normalized samples;
5. find leading covariance eigenvectors with deterministic power iteration and Gram-Schmidt deflation;
6. canonicalize each axis sign;
7. store the mean and axes as an immutable Float32 profile.

The trainer materializes a centered Float32 sample matrix but not a dimension-squared covariance matrix. Training is an offline operation and may allocate proportional to `sample_count × dimension`.

`ErrPCAConvergence` means the configured iteration limit was insufficient. Increase `MaxIterations`; do not publish a partial profile.

### Projection write path

`Project` appends to caller-owned Float32 capacity and does not grow it.

```go
values := make([]float32, 0, profile.ProjectionCount())
values, err = profile.Project(values, vector)
if err != nil {
    return err
}

for projectionNo, value := range values {
    storage := make([]byte, 0, 5)
    builder := schottky.NewBuilder(storage)
    builder.Float32(value, schottky.AscNullsLast)
    projectionKey, err := builder.Key()
    if err != nil {
        return err
    }
    // Insert profileID, projectionNo, value, projectionKey, and item ID.
}
```

A write must store the original vector and all projection rows in one transaction. A missing projection row silently lowers recall.

## Multi-projection schema

The following names and types are templates. Adapt identity, binary, timestamp, and vector-distance types to the storage engine without changing the key order.

```sql
CREATE TABLE ann_profile (
    profile_id          BIGINT      PRIMARY KEY,
    profile_version     INTEGER     NOT NULL,
    method              VARCHAR(32) NOT NULL,
    dimension_count     INTEGER     NOT NULL,
    projection_count    INTEGER     NOT NULL,
    normalization       VARCHAR(16) NOT NULL,
    profile_fingerprint BINARY(32)  NOT NULL UNIQUE,
    profile_bytes       BLOB        NOT NULL,
    created_at          TIMESTAMP   NOT NULL,
    CHECK (dimension_count > 0),
    CHECK (projection_count > 0)
);

CREATE TABLE ann_active_profile (
    partition_id BIGINT PRIMARY KEY,
    profile_id   BIGINT NOT NULL,
    FOREIGN KEY (profile_id) REFERENCES ann_profile (profile_id)
);

CREATE TABLE ann_item (
    partition_id    BIGINT NOT NULL,
    item_id         BIGINT NOT NULL,
    dimension_count INTEGER NOT NULL,
    embedding       BLOB   NOT NULL,
    PRIMARY KEY (partition_id, item_id)
);

CREATE TABLE ann_projection (
    profile_id       BIGINT     NOT NULL,
    partition_id     BIGINT     NOT NULL,
    projection_no    INTEGER    NOT NULL,
    projection_key   BINARY(5)  NOT NULL,
    projection_value REAL       NOT NULL,
    item_id          BIGINT     NOT NULL,
    PRIMARY KEY (
        profile_id,
        partition_id,
        projection_no,
        projection_key,
        item_id
    ),
    CHECK (projection_no >= 0),
    FOREIGN KEY (profile_id)
        REFERENCES ann_profile (profile_id),
    FOREIGN KEY (partition_id, item_id)
        REFERENCES ann_item (partition_id, item_id)
);
```

The primary-key order is required:

```text
profile → partition → projection number → projection key → stable item tie-break
```

It supports independent range scans for every projection while keeping one physical schema for profiles with different projection counts. A wide table with one column and index per projection is faster in some engines but makes profile replacement and variable projection counts operationally expensive.

### Storage amplification

A profile with `m` projections writes `m` rows per vector.

```text
projection rows = vector rows × projection count
```

Use 16 projections only when write amplification is acceptable. Profiles with 64–128 projections belong in read-heavy or analytical partitions, not high-churn transactional tables.

## Exact scalar top-K from a B-tree

The benchmark retrieves the `K` smallest absolute scalar differences per projection. A B-tree exposes two ordered streams around the query key:

- lower values in descending order;
- equal or higher values in ascending order.

An application can compare the current absolute differences and advance the closer cursor until it consumes exactly `K` rows. This costs approximately `m × K` scalar hits across `m` projections.

A static SQL query normally materializes up to `K` rows from each direction and then keeps the closest `K`. Its read upper bound is `2 × m × K`.

Fetching `K/2` from each direction is cheaper but is not exact scalar top-K when nearby values are concentrated on one side.

## SQL union and exact rerank

This two-projection example shows the complete static-SQL path. Generate one `axis_n_pool` and `axis_n` pair per projection.

```sql
WITH
axis_0_pool AS (
    SELECT item_id, projection_no, projection_value
    FROM (
        (
            SELECT item_id, projection_no, projection_value
            FROM ann_projection
            WHERE profile_id = :profile_id
              AND partition_id = :partition_id
              AND projection_no = 0
              AND projection_key >= :query_key_0
            ORDER BY projection_key ASC, item_id ASC
            LIMIT :per_projection_neighbors
        )
        UNION ALL
        (
            SELECT item_id, projection_no, projection_value
            FROM ann_projection
            WHERE profile_id = :profile_id
              AND partition_id = :partition_id
              AND projection_no = 0
              AND projection_key < :query_key_0
            ORDER BY projection_key DESC, item_id ASC
            LIMIT :per_projection_neighbors
        )
    ) AS two_sides
),
axis_0 AS (
    SELECT item_id, projection_no
    FROM axis_0_pool
    ORDER BY ABS(projection_value - :query_value_0), item_id
    LIMIT :per_projection_neighbors
),
axis_1_pool AS (
    SELECT item_id, projection_no, projection_value
    FROM (
        (
            SELECT item_id, projection_no, projection_value
            FROM ann_projection
            WHERE profile_id = :profile_id
              AND partition_id = :partition_id
              AND projection_no = 1
              AND projection_key >= :query_key_1
            ORDER BY projection_key ASC, item_id ASC
            LIMIT :per_projection_neighbors
        )
        UNION ALL
        (
            SELECT item_id, projection_no, projection_value
            FROM ann_projection
            WHERE profile_id = :profile_id
              AND partition_id = :partition_id
              AND projection_no = 1
              AND projection_key < :query_key_1
            ORDER BY projection_key DESC, item_id ASC
            LIMIT :per_projection_neighbors
        )
    ) AS two_sides
),
axis_1 AS (
    SELECT item_id, projection_no
    FROM axis_1_pool
    ORDER BY ABS(projection_value - :query_value_1), item_id
    LIMIT :per_projection_neighbors
),
projection_hits AS (
    SELECT item_id, projection_no FROM axis_0
    UNION ALL
    SELECT item_id, projection_no FROM axis_1
),
candidate_ids AS (
    SELECT
        item_id,
        COUNT(DISTINCT projection_no) AS collision_count
    FROM projection_hits
    GROUP BY item_id
    HAVING COUNT(DISTINCT projection_no) >= :minimum_collisions
    ORDER BY collision_count DESC, item_id
    LIMIT :candidate_cap
)
SELECT
    item.item_id,
    vector_distance(item.embedding, :query_embedding) AS exact_distance
FROM candidate_ids AS candidate
JOIN ann_item AS item
  ON item.partition_id = :partition_id
 AND item.item_id = candidate.item_id
ORDER BY exact_distance ASC, item.item_id ASC
LIMIT :result_count;
```

`vector_distance` denotes an engine function or application UDF that decodes the original vector and computes the exact configured metric. It is not a Schottky key comparison.

Set `minimum_collisions = 1` for pure OR. Higher values implement QALSH-style collision filtering. The `candidate_cap` must be no smaller than the measured candidate p95 for the selected profile; a lower cap changes recall.

For 64–128 projections, prefer application-managed two-cursor scans or generated statements over a handwritten SQL CTE with hundreds of branches.

## Query-centered range mode

The same schema supports query-centered QALSH ranges instead of fixed scalar top-K. For projection `i`, encode lower and upper Float32 bounds around the query projection:

```text
query_projection[i] - radius ≤ item_projection[i] ≤ query_projection[i] + radius
```

Scan every projection range, count distinct projection collisions, rerank exact vectors, and expand `radius` when too few candidates survive. Range mode is closer to the original QALSH algorithm; the benchmark below measures fixed scalar top-K because it gives a comparable row budget across projections.

## Tuned operating points

The measurements use a 10,000-row, 1,536-dimensional, unit-normalized public text-embedding sample. A fixed split produced 9,000 corpus rows and 500 held-out queries. Ground truth is exact cosine top-10. Recall@10 is the fraction of exact neighbors present after candidate extraction and exact reranking.

### Trained PCA

PCA was trained only on the 9,000 corpus rows. One axis was more selective than multi-axis PCA at the measured 90–95% operating band.

| PCA axes | Scalar neighbors per axis | Mean candidates | Corpus reranked | Recall@10 |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 4,250 | 4,250 | 47.2% | 90.64% |
| 1 | 4,500 | 4,500 | 50.0% | 91.64% |
| 1 | 5,000 | 5,000 | 55.6% | 93.60% |
| 1 | 6,000 | 6,000 | 66.7% | 96.08% |

Recommended measured starting points:

- target near 90%: one PCA axis, `K = ceil(0.47 × partition_rows)`;
- target near 94%: one PCA axis, `K = ceil(0.56 × partition_rows)`.

These fractions are initial values for similar distributions, not portable guarantees.

### Training-free Gaussian OR

| Projection indexes | Scalar neighbors per projection | Scalar hits consumed by two-cursor merge | Mean candidates | Corpus reranked | Recall@10 |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 16 | 1,000 | 16,000 | 7,623 | 84.7% | 93.32% |
| 28 | 500 | 14,000 | 7,175 | 79.7% | 90.28% |
| 60 | 250 | 15,000 | 7,329 | 81.4% | 91.54% |
| 140 | 100 | 14,000 | 7,107 | 79.0% | 90.12% |

Static two-sided SQL may read up to twice the listed scalar-hit count. More projections did not remove the need to rerank most rows.

### Gaussian collision filtering

| Projection indexes | Neighbors per projection | Minimum collisions | Scalar hits consumed by two-cursor merge | Mean candidates | Corpus reranked | Recall@10 |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 64 | 1,000 | 6 | 64,000 | 6,517 | 72.4% | 94.68% |
| 128 | 500 | 6 | 64,000 | 6,446 | 71.6% | 94.22% |
| 128 | 1,000 | 14 | 128,000 | 5,055 | 56.2% | 93.36% |

For the last configuration, threshold 15 reduced candidates to 45.3% but also reduced Recall@10 to 89.30%. Collision filtering improves candidate selectivity at the cost of substantially more index reads and storage.

## Benchmark figures

![Candidate coverage after scalar projection retrieval](assets/vector-ann-recall-candidates.png)

The collision curve reports exact-rerank candidates, not index-read cost. Collision profiles read every configured per-projection hit before filtering.

![Training-free Gaussian OR parameter sweep](assets/vector-ann-qalsh-tuning.png)

The white contour is 90% Recall@10; the dark contour is 95%. The source rows for both figures are in [`vector-ann-benchmark.csv`](assets/vector-ann-benchmark.csv).

### Reproducibility metadata

| Property | Value |
| --- | --- |
| source rows | 10,000 |
| dimensions | 1,536 |
| corpus rows | 9,000 |
| held-out queries | 500 |
| query count used in each mean | 500 |
| exact result count | 10 |
| split and projection seed | `20260830` |
| vector subset SHA-256 | `6ea192c75d72419cfcc53f5c77175dbdda2d2ad644544bb653931176ace0fcff` |
| projection-key precision | Float32 |
| PCA training | exact covariance eigendecomposition for the benchmark |
| Gaussian retrieval | exact absolute scalar top-K before OR or collision filtering |

The experiment measures candidate recall in memory. It does not measure storage-engine latency, buffer-cache behavior, transaction contention, or network round trips.

## OLTP operation

- Partition by the tenant or business predicate used in every query; tune against the post-filter row count.
- Prefer exact scan for small partitions.
- Prefer one-axis PCA when a representative training sample and profile rebuild are acceptable.
- Limit training-free profiles to 16 projections on write-sensitive tables.
- Insert the item and all projection rows atomically.
- Keep a stable item ID in the index to make Float32 ties deterministic.
- Record candidate count, scalar hits, and exact-distance calls per query.

Do not advertise 90% recall from the 16-projection preset without a held-out evaluation. Its measured recall required reranking 84.7% of the sample corpus.

## HTAP operation

- Train PCA from a stable analytical snapshot, not an uncommitted transactional scan.
- Backfill a new profile ID without mutating active projection rows.
- Dual-write active and replacement profiles during a long backfill.
- Validate recall and candidate p95 before switching the active profile.
- Retire the old profile only after readers stop referencing it.
- Use 64–128 Gaussian projections only when read concurrency, storage amplification, and index fan-out have been measured.

## Drift and profile replacement

Retrain PCA when held-out recall falls below the schema target or the input distribution changes materially. Replacing the mean or any axis changes every projection key.

A clean replacement sequence is:

1. train and serialize a new profile;
2. insert a new immutable profile row;
3. backfill its projection rows;
4. evaluate held-out Recall@K and candidate p95;
5. start dual writes;
6. atomically switch the active profile ID;
7. drain old readers;
8. delete obsolete projection rows.

Never overwrite `profile_bytes` under an existing profile ID.
