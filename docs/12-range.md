# Range and multirange values

## Canonical range

Canonicalize bounds with the same rules as the owning SQL range type before encoding. Discrete ranges should normally use the database's canonical form, such as converting equivalent inclusive and exclusive integer bounds to one representation.

Build a range as a nested tuple:

```text
(class, lower_kind, lower_value?, upper_kind, upper_value?)
```

Recommended ranks are:

| Component | Rank order |
| --- | --- |
| class | empty, non-empty |
| lower kind | negative infinity, inclusive, exclusive |
| upper kind | exclusive, inclusive, positive infinity |

Encode finite bounds with the element profile. Embed the nested key with `Builder.Tuple`.

The rank table defines Schottky range order. If exact parity with a database range comparator is required, use that comparator's rank and canonicalization rules.

## Multirange

Canonicalize to sorted, non-overlapping, non-adjacent ranges. Encode each canonical range as one nested tuple element inside a second nested key. Equal mathematical unions then produce equal keys.

## Validity

Reject:

- a lower bound greater than its upper bound;
- an invalid empty/non-empty combination;
- a bound encoded with a different element schema;
- overlapping or unsorted multirange members after canonicalization is expected.

Canonicalization is outside Schottky because adjacency and successor rules belong to the element domain.
