# Temporal values

## Scalar representations

Temporal payloads use the signed integer transform at the stated width:

| Domain | Canonical scalar |
| --- | --- |
| date | signed days since `2000-01-01`, `int32` |
| time | microseconds since `00:00:00`, `int64` |
| timestamp without zone | microseconds since `2000-01-01 00:00:00`, `int64` |
| timestamp instant | microseconds since `2000-01-01 00:00:00 UTC`, `int64` |
| elapsed duration | signed microseconds, `int64` |

`DateNegativeInfinity`, `DatePositiveInfinity`, `TimestampNegativeInfinity`, and `TimestampPositiveInfinity` are the ordered infinity sentinels. The finite date domain is `[-2_451_545, 2_145_031_949)`. The finite timestamp domain is `[-211_813_488_000_000_000, 9_223_371_331_200_000_000)`.

## Validation

- Time-of-day is in `[0, 86_400_000_000]`; the inclusive endpoint represents `24:00:00`.
- Leap seconds must follow one schema-wide normalization rule.
- Civil timestamps must use one calendar and gap/overlap policy.
- Zoned instants must be converted to UTC before encoding.
- Monotonic-clock metadata is never persisted.

## Time with numeric offset

`ZonedTime` stores local microseconds and a UTC offset in seconds measured eastward. It accepts offsets through `±15:59:59` and compares:

```text
(local_microseconds - east_offset_seconds * 1_000_000,
 -east_offset_seconds)
```

The primary value is not reduced modulo one day. The second component distinguishes equal UTC-equivalent values by their original numeric offset.

## Calendar intervals

`IntervalOrderValue(months, days, microseconds)` computes an exact signed `Int128` scalar:

```text
microseconds + 86_400_000_000 * (days + 30 * months)
```

Encode the result with `Builder.Int128`. Distinct component triples can compare equal; one month and thirty days intentionally produce the same key. Use `Duration` instead when the domain is already an elapsed-microsecond scalar.

The all-minimum and all-maximum interval triples map to the two `Int128` extrema. Mixing interval comparison profiles in one keyspace is invalid.
