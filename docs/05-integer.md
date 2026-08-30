# Integer and ranked values

## Unsigned integers

Encode the value in big-endian order at its declared width. Supported unsigned widths are 8, 16, 32, and 64 bits. Numeric order then equals unsigned bytewise order.

## Signed integers

Flip the most-significant sign bit, then encode in big-endian order at the declared width:

```text
ordered = unsigned_bits(value) XOR sign_mask
```

This rotates two's-complement order so the minimum signed value starts at zero, zero starts at the sign mask, and the maximum value ends at all ones.

Integer width is part of the schema. Signed encoders support 8, 16, 32, 64, and 128 bits; `Int128` uses `{High int64, Low uint64}` two's-complement limbs. Encodings at different widths are not interchangeable even when they represent the same number.

## Descending order

Complement the complete fixed-width ascending payload.

## SQL mappings

| SQL domain | Encoding |
| --- | --- |
| `smallint`, `smallserial` | `Int16` |
| `integer`, `serial` | `Int32` |
| `bigint`, `bigserial` | `Int64` |
| non-negative counters and sequence values | matching `Uint` width |
| `money` with fixed currency and scale | signed minor units |
| enum | immutable unsigned rank |
| object identifier types | declared unsigned identifier width |
| `pg_lsn` | `Uint64` |

Enum labels are not encoded directly. Persist a stable rank table; inserting a new label between existing labels requires re-encoding later ranks.
