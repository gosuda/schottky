# Binary string

## Domain

A binary string is an uninterpreted sequence of bytes supplied as a Go `string`. Ordering is unsigned bytewise lexicographic order. No Unicode validation, normalization, case folding, locale rule, or trailing-space rule is applied.

Use this profile for SQL text only when the index contract is explicitly binary. Use [collation strings](02-collation-string.md) for locale-sensitive text.

## Ascending payload

Encode each source byte in order:

| Source byte | Encoded bytes |
| --- | --- |
| `00` | `00 ff` |
| `01` through `ff` | unchanged |

Append `00 00` after the final source byte. The terminator sorts before every escaped or nonzero continuation, so a string sorts before every string for which it is a proper prefix.

The empty payload is `00 00`.

## Descending payload

Bitwise-complement every byte of the ascending payload. The terminator becomes `ff ff`; an escaped zero becomes `ff 00`. Complementing the complete payload reverses all unequal byte comparisons, including prefix comparisons.

## Field envelope

The enclosing field adds one presence tag before the payload. Direction never changes the presence tag; null placement and value direction remain independent. See [key layout](03-key-layout.md).

## Size

For an input of length $n$ containing $z$ zero bytes, payload length is $n+z+2$. Total field length is $n+z+3$.

Encoding and decoding accept caller-owned storage. A short destination fails without partially appending the field.
