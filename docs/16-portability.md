# Portability and evolution

## Comparison primitive

Indexes must compare keys as unsigned bytes. Signed-byte comparison, locale-aware string comparison, C-string termination, and transport encodings such as Base64 do not preserve the contract unless the storage engine explicitly applies unsigned binary order to decoded bytes.

## Schema identity

A key schema includes:

- ordered field list and field widths;
- null placement and direction per field;
- decimal, float, temporal, network, and spatial policies;
- collation and external token producer versions;
- normalization and tie-breaker rules;
- nested array, range, record, and extension schemas.

Schottky keys carry no self-description. Store a schema identifier in index metadata or prepend a caller-owned version discriminator.

## Clean migrations

Any ordering or equality change requires rebuilding affected keys. Do not mix old and new encodings in one ordered keyspace. A dual-read migration may maintain separate keyspaces, but each keyspace must remain internally uniform.

## Architecture

The format is independent of host endianness and word size. Fixed numbers are emitted in network byte order. SIMD and scalar builds produce identical bytes.

## Resource limits

The core builder never allocates or grows its destination. Callers choose maximum key, nested scratch, and decoded-value sizes. Treat `ErrShortBuffer` as a capacity decision, not malformed input.

A service accepting untrusted values should cap source lengths before scanning them for exact encoded size. Decoders reject truncation, invalid escapes, invalid class markers, and non-canonical fixed payloads where the type defines them.
