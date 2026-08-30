# Go API

## Builder lifecycle

```go
storage := make([]byte, 0, 128)
b := schottky.NewBuilder(storage)
b.Int64(42, schottky.AscNullsLast)
prefixLen := b.Len()
b.String("Ada", schottky.DescNullsFirst)
key, err := b.Key()
```

`NewBuilder` borrows the supplied slice and appends only within its capacity. It does not retain any other reference. Each field method is atomic and records the first error. `Reset` reuses a builder with another caller-owned slice.

`Len` records a safe field boundary only when called between field operations. `Key` returns the borrowed slice and sticky error. The returned key aliases the supplied storage.

## Capacity planning

Fixed-size constants include the one-byte presence tag. `EncodedBytesSize`, `EncodedStringSize`, and `EncodedDecimalSize` return exact field sizes. `MaxEncodedBinarySize` returns the zero-scan upper bound $2n+3$.

Size functions do not allocate. A builder still checks capacity so stale or untrusted size calculations cannot write beyond the supplied buffer.

## Ordering constants

The four closed `Order` values combine value direction and null placement:

- `AscNullsFirst`
- `AscNullsLast`
- `DescNullsFirst`
- `DescNullsLast`

A value outside this set produces `ErrInvalidOrder`.

## Decoder lifecycle

A decoder consumes fields in schema order. Fixed-width methods return typed values. `Decoder.Bytes` writes an unescaped value into caller-owned capacity. Null is reported as `Null`; malformed, truncated, or non-canonical input records a sticky error.

A successful decode should end with `Remaining() == 0`. Trailing bytes otherwise represent additional fields or a schema mismatch.

## Aliases

The root package exposes semantic methods where their representation and validation are stable:

- binary string, bytes, collation key, and nested tuple;
- Boolean, signed and unsigned integers, floats, and decimal text;
- date, time, timestamp, duration, enum rank, LSN, UUID, MAC, IP, and network prefix.

Collections and policy-heavy database types are assembled from nested tuples. The `internal` tree contains escaping and byte transforms; callers depend only on the root package.

## Concurrency and overlap

A builder or decoder is single-owner mutable state. Separate instances are independent. Variable-width source and destination storage must not overlap. Fixed values are copied by value.
