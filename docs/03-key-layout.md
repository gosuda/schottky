# Key layout

## Composite key

A key is the concatenation of schema-known fields. It has no version byte, type tags, field count, or terminal trailer. The index schema supplies field types, widths, directions, null placement, and external normalization policies.

This omission is deliberate:

- every completed field prefix is also a literal byte prefix of every key that extends it;
- fixed-width values pay one byte for null handling and no framing overhead;
- decoding needs no dynamic dispatch;
- callers can prepend a table or index discriminator outside Schottky.

Persist the schema or a schema version beside the index. Never decode a key with a different schema.

## Presence tag

Every field starts with one byte:

| Null policy | Null | Present |
| --- | ---: | ---: |
| `NullsFirst` | `00` | `01` |
| `NullsLast` | `01` | `00` |

The payload follows only for a present value. Direction transforms the payload, not the presence tag. Therefore `DESC NULLS FIRST`, `DESC NULLS LAST`, and both ascending variants are independent.

## Direction

An ascending payload is the canonical type encoding. A descending payload is its bytewise complement. All type profiles reserve complete payload boundaries, so complementing reverses both ordinary and proper-prefix comparisons.

## Tuple order

Given one schema, unsigned bytewise comparison of complete keys equals lexicographic comparison of field values. The first unequal field decides the result. If every field in one key equals the corresponding field in another, the shorter key sorts first.

A schema should normally produce a fixed field count. Variable field counts are valid only when the shorter-tuple-first rule is intentional.

## Failure and ownership

`Builder` appends to caller-owned capacity and never grows it. A field is atomic: invalid input or insufficient capacity leaves the key at its previous length and records a sticky error. `Decoder` borrows the key and writes variable-width results into caller-owned buffers.

Builders and decoders are not safe for concurrent mutation. Independent instances may share immutable source values.
