# Schottky

Schottky encodes typed values and composite tuples so unsigned bytewise key order matches the declared value order.

## Properties

- Caller-owned encoding and decoding buffers
- Allocation-free hot path when capacity is sufficient
- Independent ascending or descending order and null placement per field
- Literal complete-field prefixes for prefix filters and range scans
- Scalar and nested profiles for modern SQL data
- No third-party dependencies
- Optional Go 1.27 portable SIMD byte transforms

## Example

```go
storage := make([]byte, 0, 128)
builder := schottky.NewBuilder(storage)
builder.Int64(42, schottky.AscNullsLast)
accountPrefix := builder.Len()
builder.String("Ada", schottky.DescNullsFirst)

key, err := builder.Key()
if err != nil {
	return err
}
prefix := key[:accountPrefix]
```

`Builder` never grows `storage`. Insufficient capacity returns `schottky.ErrShortBuffer` without partially appending a field.

Start with the [key layout](docs/03-key-layout.md), [Go API](docs/18-api.md), [SQL type map](docs/17-sql-type-map.md), [format compatibility guide](docs/19-format-compatibility.md), and [utf8mb4 collation map](docs/20-utf8mb4-map.md).

Enable the experimental SIMD path with `GOEXPERIMENT=simd` on Go 1.27. Scalar and SIMD builds emit identical keys.
