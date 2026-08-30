# Document and search values

## JSON values

Textual JSON has no canonical ordering. Choose one of these profiles:

- encode normalized source bytes when source-text order is intentional;
- encode a versioned canonical JSON representation for structural equality;
- encode a database-produced sortable token for exact database comparator parity.

A canonical structural profile must define object-key order, duplicate keys, number normalization, string normalization, escape handling, array order, and class order among null, Boolean, number, string, array, and object. Schottky does not silently choose these policies.

## XML

Use canonical XML bytes from a named canonicalization version, or an application-specific ordered token. Record namespace, comment, whitespace, encoding, and entity-expansion policies. Raw source bytes are valid only when source order is the intended contract.

## Full-text search

Native full-text vectors and queries depend on parser, dictionary, collation, weight, and normalization behavior. Use a database-produced token or a canonical nested tuple whose producer and configuration are versioned with the index.

## Transaction snapshots

Encode a snapshot only after canonicalizing member transaction identifiers into a stable sorted sequence. Use a nested tuple for scalar bounds and active identifiers. Deprecated and current snapshot domains should not share a schema unless their equality and width rules are identical.

## API

Pass canonical tokens to `Builder.Bytes`, `Builder.CollationKey`, or `Builder.Tuple`. Schottky preserves token order; it does not validate the external grammar.
