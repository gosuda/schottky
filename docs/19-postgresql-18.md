# PostgreSQL 18 compatibility

This mapping targets PostgreSQL 18 B-tree comparison. The Go API remains vendor-neutral; `pg18` is schema metadata, not a runtime mode.

## ORDER BY options

Use the `Order` value matching the complete SQL clause:

| SQL | Schottky |
| --- | --- |
| `ASC` or `ASC NULLS LAST` | `AscNullsLast` |
| `ASC NULLS FIRST` | `AscNullsFirst` |
| `DESC` or `DESC NULLS FIRST` | `DescNullsFirst` |
| `DESC NULLS LAST` | `DescNullsLast` |

PostgreSQL defaults nulls last for ascending order and nulls first for descending order. Do not infer null placement from direction in application code; store the selected `Order` in the key schema.

For a nested array, range, multirange, or record, build internal components in ascending comparator order and apply the SQL direction only to the outer `Tuple` field.

## Type options

| PostgreSQL type | Required encoding |
| --- | --- |
| `boolean` | `Bool` |
| `smallint`, `integer`, `bigint` | matching signed integer width |
| `real`, `double precision` | `Float32`, `Float64` |
| `numeric`, `decimal` | `Decimal` |
| `money` | `Int64` minor-unit value |
| `bytea` | `Bytes` |
| `uuid` | `UUID` |
| `macaddr`, `macaddr8` | `MAC48`, `MAC64` |
| `pg_lsn` | `LSN` |
| enum | immutable catalog sort rank through `Enum` |
| `date` | `Date`; days from `2000-01-01` |
| `time` | `Time`; microseconds in `[0, 86_400_000_000]` |
| `timestamp`, `timestamptz` | `Timestamp`; microseconds from `2000-01-01`, with UTC conversion for `timestamptz` |
| `timetz` | `ZonedTime` with east-positive `UTCOffsetSeconds` |
| `interval` | `Int128(IntervalOrderValue(months, days, microseconds))` |
| `inet` without a netmask | `IP` |
| `inet` with a netmask | `IPPrefix`; preserve host bits |
| `cidr` | `NetworkPrefix`; reject host bits |
| `bit`, `varbit` | packed-byte tuple from [network and bit values](10-network-bit.md) |
| array, record | nested layout from [structured values](11-structured.md) |
| range, multirange | nested layout from [range values](12-range.md) |

`DateNegativeInfinity`, `DatePositiveInfinity`, `TimestampNegativeInfinity`, and `TimestampPositiveInfinity` map the native temporal infinities. The finite domain checks reject scalar values PostgreSQL cannot store.

IPv4-mapped IPv6 addresses remain IPv6. IPv6 zones are rejected. `NetworkPrefix` rejects a non-canonical value instead of silently clearing host bits.

## Text and collation

For UTF-8 `COLLATE "C"` or `COLLATE "POSIX"`, use `String` after applying the SQL type's trailing-space rule.

For a deterministic locale collation, encode two ascending nested components:

```text
(collation_key, original_UTF8_bytes)
```

The second component is the original source bytes, not normalized text. PostgreSQL uses that bytewise tie-break after the collation provider reports equality. For a nondeterministic collation, omit the source tie-break because collation equality is intentional.

A provider name alone is insufficient schema identity. Record provider, provider version, locale, deterministic flag, strength/options, and the exact key producer. Rebuild keys after any provider-version or option change.

## Caller-supplied tokens

Keep JSON binary values, text-search vectors and queries, physical tuple identifiers, transaction identifiers with wraparound semantics, unknown extension operator classes, and catalog-state-dependent labels behind versioned caller-supplied canonical tokens. Their native comparators depend on private representation, database state, or a separate collation profile; presenting them as portable scalar encodings would be misleading.

## Migration

The network, date, timestamp, array, and range layouts in this profile differ from earlier Schottky documentation. Do not mix them in one ordered keyspace. Assign a new schema identifier and rebuild affected keys.
