# Temporal values

## Scalar representations

Temporal payloads use the signed integer transform at the stated width:

| Domain | Canonical scalar |
| --- | --- |
| `date` | signed days since 1970-01-01, `int32` |
| `time` | microseconds since 00:00:00, `int64` |
| `timestamp` | microseconds since 1970-01-01 00:00:00 in the schema's civil calendar, `int64` |
| `timestamptz` | microseconds since the Unix epoch in UTC, `int64` |
| elapsed duration | signed microseconds, `int64` |

Microseconds cover PostgreSQL 18 precision and its timestamp range. The epoch choice does not affect order if subtraction cannot overflow the declared representation.

## Validation

- Time-of-day is in `[0, 86_400_000_000)`.
- Leap seconds must follow one schema-wide normalization rule.
- Civil timestamps must use one calendar and gap/overlap policy.
- Zoned instants must be converted to UTC before encoding.
- Monotonic-clock metadata is never persisted.

## Time with time zone

Normalize `timetz` to the database's comparison scalar before encoding. Preserve an original offset only as a later tie-breaker when the database distinguishes equal instants by offset.

## Calendar intervals

Months, days, and microseconds do not have a database-independent total duration. Choose one profile per index:

- **elapsed:** convert to signed microseconds with an explicit overflow policy;
- **calendar tuple:** encode `(months, days, microseconds)` as three signed fields;
- **database comparator:** compute the database's canonical comparison scalar and encode that scalar.

The profile is schema metadata. Mixing profiles produces invalid ordering.

## Infinity

Databases that support temporal infinities should prepend a class field: `0` for negative infinity, `1` for finite, `2` for positive infinity. The finite scalar follows only for class `1`.
