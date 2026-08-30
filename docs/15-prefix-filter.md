# Prefix filters and scan bounds

## Field prefixes

A completed leading field sequence is a literal byte prefix of every composite key that extends it. Record `Builder.Len()` immediately after the last constrained field and use `key[:n]` as the filter prefix.

Never cut inside a field. An interior prefix can end in an escape sequence, omit a fixed-width suffix, or merge unrelated values.

## Bloom filter extraction

A prefix Bloom filter may hash:

- a table or index discriminator prepended by the caller;
- one complete leading field;
- any fixed count of complete leading fields.

The extraction policy is index metadata. Writers and readers must use the same field count and key-schema version. Very short or low-cardinality prefixes can increase false positives without improving pruning.

## Range bound

For byte prefix `p`, the half-open scan interval is:

```text
[p, PrefixUpperBound(p))
```

`PrefixUpperBound` finds the final byte not equal to `ff`, increments it, and discards the suffix. No finite upper bound exists when every byte is `ff`; use an unbounded endpoint.

The function copies into caller-owned storage and fails atomically when capacity is insufficient.

## Descending fields

Prefix behavior is unchanged for descending fields because the direction transform stays inside each complete field. Build the constrained values with the index's actual directions; do not reverse or complement a completed prefix again.
