# SQL type map

This map covers general-purpose SQL types and common application or extension domains. “Adapter” means the database or application must supply canonical semantics before Schottky encodes the result.

| SQL type or domain | Profile |
| --- | --- |
| `smallint`, `integer`, `bigint` | signed integer at matching width |
| unsigned dialect integers | unsigned integer at matching width |
| `numeric`, `decimal` | decimal text |
| `real`, `double precision` | canonical float |
| fixed-currency `money` | signed minor units; currency belongs to schema |
| `boolean` | Boolean |
| binary SQL values | binary bytes |
| binary `text`, `varchar`, `char` | binary string after domain padding rules |
| collated character types | versioned collation profile; raw source-byte tie-break for deterministic equality |
| date, time, and timestamp variants | validated temporal scalar from the `2000-01-01` profile epoch |
| zoned time | `ZonedTime` |
| `interval` | `Int128(IntervalOrderValue(...))` |
| `uuid`, ULID | canonical 16 bytes |
| host address with optional prefix | `IP` or host-preserving `IPPrefix` |
| canonical network prefix | `NetworkPrefix` |
| 48-bit or 64-bit MAC address | canonical 6 or 8 bytes |
| fixed or varying bit string | packed bits and significant length |
| enum | immutable unsigned rank |
| domain | normalized base profile |
| array | nested element tuple |
| composite or row | nested attribute tuple |
| range, multirange | canonical nested range tuples |
| `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle` | schema-named spatial tuple |
| structured document formats | adapter-produced canonical token |
| native full-text vectors or queries | database-produced token or versioned tuple |
| log sequence number | unsigned 64-bit scalar |
| transaction snapshot | canonical nested snapshot tuple |
| object identifiers and catalog aliases | declared unsigned identifier profile |
| vectors, GIS, extension types | operator-class token or explicit nested tuple |

Not every native database comparator is portable or documented as a wire contract. For exact index parity, test the adapter against database `ORDER BY`, equality, null, special-value, and collation behavior before persisting keys.
