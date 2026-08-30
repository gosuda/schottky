# Range and multirange values

## Canonical range

Canonicalize bounds with the owning range subtype before encoding. A discrete subtype must convert equivalent representations to one canonical form. Empty sorts before every non-empty range.

Build each non-empty range from three nested keys:

```text
lower = (kind, Tuple(value)?, inclusion?)
upper = (kind, Tuple(value)?, inclusion?)
range = (class, Tuple(lower)?, Tuple(upper)?)
```

Use these unsigned ranks:

| Component | Rank |
| --- | ---: |
| empty class | `0` |
| non-empty class | `1` |
| unbounded lower kind | `0` |
| finite lower kind | `1` |
| finite upper kind | `0` |
| unbounded upper kind | `1` |
| inclusive finite lower | `0` |
| exclusive finite lower | `1` |
| exclusive finite upper | `0` |
| inclusive finite upper | `1` |

For a finite bound, encode the element value before its inclusion rank. The subtype value uses its ascending profile and cannot be null. This produces `negative infinity < finite < positive infinity`; at the same finite value, an inclusive lower precedes an exclusive lower, while an exclusive upper precedes an inclusive upper.

Apply descending order only to the outer range `Tuple`. Bounds inside the range remain ascending.

## Multirange

Canonicalize to sorted, non-overlapping, non-adjacent ranges. Encode each canonical range as one nested tuple element inside a second nested key. Equal mathematical unions then produce equal keys.

## Validity

Reject:

- a lower bound greater than its upper bound;
- an invalid empty/non-empty combination;
- a bound encoded with a different element schema;
- overlapping or unsorted multirange members after canonicalization is expected.

Canonicalization is outside Schottky because adjacency and successor rules belong to the element domain.
