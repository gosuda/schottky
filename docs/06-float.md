# Floating point

## Equality policy

The canonical profile follows database-oriented total ordering:

- all NaN bit patterns compare equal and sort after positive infinity;
- negative zero and positive zero compare equal and share one encoding;
- infinities retain IEEE order;
- other values retain numeric order.

This matches PostgreSQL's documented treatment of NaN and avoids distinct keys for values that SQL equality commonly treats as equal. A domain requiring IEEE `totalOrder` needs a separate profile.

## Ascending transform

For a non-NaN IEEE bit pattern:

1. replace either zero with positive zero;
2. if the sign bit is set, complement every bit;
3. otherwise, flip only the sign bit;
4. emit the result in big-endian order.

Encode every NaN as all one bits. Use four bytes for `Float32` and eight bytes for `Float64`.

## Descending order

Complement the complete ascending payload after canonicalization.

## Consequences

The transform is reversible for every canonical non-NaN value. Decoding the all-ones sentinel returns a canonical quiet NaN. Original NaN sign and payload, and the sign of zero, are intentionally discarded.
