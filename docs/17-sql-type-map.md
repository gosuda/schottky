# SQL type map

This map covers PostgreSQL 18 general-purpose types and common SQL or extension domains. “Adapter” means the database or application must supply canonical semantics before Schottky encodes the result.

| SQL type or domain | Profile |
| --- | --- |
| `smallint`, `integer`, `bigint`, serial aliases | signed integer at matching width |
| unsigned dialect integers | unsigned integer at matching width |
| `numeric`, `decimal` | decimal text |
| `real`, `double precision` | canonical float |
| fixed-currency `money` | signed minor units; currency belongs to schema |
| `boolean` | Boolean |
| `bytea`, binary SQL types | binary bytes |
| binary `text`, `varchar`, `char` | binary string after domain padding rules |
| collated character types | versioned collation profile; raw source-byte tie-break for deterministic equality |
| `date`, `time`, `timestamp`, `timestamptz` | validated temporal scalar from the `2000-01-01` epoch |
| `timetz` | `ZonedTime` |
| `interval` | `Int128(IntervalOrderValue(...))` |
| `uuid`, ULID | canonical 16 bytes |
| `inet` | `IP` or host-preserving `IPPrefix` |
| `cidr` | canonical `NetworkPrefix` |
| `macaddr`, `macaddr8` | canonical 6 or 8 bytes |
| `bit`, `varbit` | packed bits and significant length |
| enum | immutable unsigned rank |
| domain | normalized base profile |
| array | nested element tuple |
| composite or row | nested attribute tuple |
| range, multirange | canonical nested range tuples |
| `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle` | schema-named spatial tuple |
| `json`, `jsonb`, `xml` | adapter-produced canonical token |
| `tsvector`, `tsquery` | database-produced token or versioned tuple |
| `pg_lsn` | unsigned 64-bit scalar |
| `pg_snapshot`, `txid_snapshot` | canonical nested snapshot tuple |
| OID and `reg*` aliases | declared unsigned identifier profile |
| vectors, GIS, extension types | operator-class token or explicit nested tuple |

Not every native database comparator is portable or documented as a wire contract. For exact index parity, test the adapter against database `ORDER BY`, equality, null, special-value, and collation behavior before persisting keys.
