# Null and boolean

## Null

Null has only the field presence tag described in [key layout](03-key-layout.md). It has no payload. Null placement is part of the field schema and must match for every key in an index.

A nullable value uses the same present-value encoding as its non-nullable counterpart. SQL wrappers such as `database/sql.NullInt64` are handled by choosing `Builder.Null` or the corresponding value method; they do not require a separate wire format.

## Boolean

A present boolean payload is one byte:

| Value | Ascending payload |
| --- | ---: |
| `false` | `00` |
| `true` | `01` |

Descending order complements the payload to `ff` and `fe`, respectively. The presence tag remains unchanged.

The decoder rejects every ascending payload byte other than `00` and `01`.
