# Decimal

## Accepted values

The decimal profile accepts canonical or non-canonical decimal text and emits one representation per numeric value. Supported syntax is:

```text
[+-]? digits [ . digits ] [ e [+-]? digits ]
[+-]? . digits [ e [+-]? digits ]
Infinity
-Infinity
NaN
```

At least one digit is required. Leading and trailing coefficient zeros, a leading plus, exponent spelling, and negative zero do not affect the key. The adjusted base-10 exponent must fit `int32` after normalization.

## Classes

The first ascending payload byte defines class order:

| Value class | Byte |
| --- | ---: |
| negative infinity | `00` |
| negative finite | `01` |
| zero | `02` |
| positive finite | `03` |
| positive infinity | `04` |
| NaN | `05` |

All NaNs share one encoding.

## Finite magnitude

Normalize a nonzero absolute value to significant digits $d_0\ldots d_{n-1}$ with no leading or trailing zero. If the value is `digits × 10^e`, define:

$$
adjusted = e+n-1
$$

Encode the magnitude as:

1. the adjusted exponent using the signed `Int32` transform without a field envelope;
2. each decimal digit as one byte from `01` through `0a`;
3. a `00` terminator.

A positive finite value stores this magnitude unchanged after class `03`. A negative finite value stores its bytewise complement after class `01`, reversing magnitude order.

The field's descending direction complements the complete class-and-magnitude payload once more.

## SQL mappings

Use this profile for `numeric` and `decimal` when arbitrary precision and PostgreSQL-style special values are required. Fixed-scale `money` values are smaller and faster as signed minor-unit integers. A database with a different NaN or infinity order requires an explicit adapter profile.
