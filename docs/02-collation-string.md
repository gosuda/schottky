# Collation string

## Contract

Schottky consumes a collation key produced by the database, ICU, or another authoritative collation engine. It does not implement Unicode normalization or locale tables. This keeps persisted ordering independent of a hidden runtime locale and avoids a mandatory collation dependency.

Encode the collation key with the [binary-bytes profile](01-bin-bytes.md). Bytewise key order then matches the order defined by that collation-key producer.

## Identity tie-breaker

Some collations assign the same key to distinct strings. Append a binary-string field containing the normalized source text when the index requires a strict total order:

```text
(collation_key ASC, normalized_source ASC)
```

Do not add the tie-breaker when collation equality is intended to collapse those strings.

## Versioning

A persisted schema must record:

- producer and version;
- locale and collation identifier;
- strength, case, accent, numeric, and normalization options;
- source normalization policy;
- tie-breaker policy.

A change to any item is a key-format migration. Mixed versions cannot share an ordered index or prefix filter.

## SQL mappings

- `text`, `varchar`, and `char`: use a database-compatible collation key when SQL collation order is required.
- `citext`-style domains: use a key generated with the domain's equality and ordering rules.
- Fixed-width character values: apply the database's trailing-space semantics before key generation.

`Builder.CollationKey` treats the supplied bytes as opaque and performs no validation.
