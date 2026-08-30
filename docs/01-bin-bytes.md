# Binary bytes

## Domain

Binary bytes cover SQL `bytea`, opaque identifiers, externally canonicalized documents, and nested Schottky keys. Ordering is unsigned bytewise lexicographic order.

## Encoding

The payload uses the same zero escaping and terminator as [binary strings](00-bin-string.md):

- `00` becomes `00 ff`.
- Every other byte is copied unchanged.
- `00 00` terminates an ascending payload.
- Descending order bitwise-complements the complete ascending payload.

The field envelope contributes one presence tag.

## Nested keys

An encoded composite key can be embedded as one binary field. Escaping preserves the nested key's bytewise order while making its end unambiguous. This is the container primitive for arrays, ranges, and user-defined records.

Build the nested key in separate caller-owned scratch storage, then pass it to `Builder.Tuple`. Source and destination must not overlap.

## Canonical input

Opaque data must have one canonical byte representation per equal value. Format-specific aliases, redundant padding, or multiple document serializations otherwise produce distinct keys. Schottky does not canonicalize external formats.

## Capacity

The exact total field size is $n+z+3$, where $n$ is input length and $z$ is its zero-byte count. The allocation-free upper bound is $2n+3$.
