# Collation string

## Algorithms

`Collator` binds one immutable, vendor-neutral `CollationProfile`:

| Algorithm | Contract |
| --- | --- |
| `UCACollation` | Unicode 14.0.0 root table plus an optional built-in tailoring |
| `SimpleCaseCollation` | Unicode 14 simple uppercase scalar order |
| `BinaryCollation` | valid UTF-8 byte order |

`UnicodeCollationVersion` identifies the public Unicode data. `CollationProfileVersion` identifies the complete versioned key behavior. Persist both. The implementation performs no normalization and rejects malformed UTF-8.

The root contract is target-versioned rather than a generic Unicode library default: precomposed Hangul uses implicit weights, one scalar emits at most eight collation elements per level, and the Tangut-special implicit range ends at U+18D7F. U+18D80 through U+18D8F use ordinary implicit weights.

The UCA profile always includes primary weights. `AccentSensitive` adds secondary weights. `CaseSensitive` adds tertiary weights independently, so all four sensitivity combinations are representable. UCA uses non-ignorable variable weighting.

`SimpleCaseCollation` has one fixed option set: `SpacePadding`, `AccentSensitive: true`, and `CaseSensitive: false`. It performs one-to-one simple uppercase mapping without expansions, contractions, or normalization.

`BinaryCollation` permits `NoPadding` or `SpacePadding` and rejects sensitivity options.

## Tailorings and defaults

The zero-value profile is root UCA, `NoPadding`, accent-insensitive, and case-insensitive. Named UCA tailorings are Croatian, Czech, Danish, Esperanto, Estonian, German phonebook, Hungarian, Icelandic, Latvian, Lithuanian, Persian, Polish, Roman/Latin, Romanian, Sinhala, Slovak, Slovenian, Spanish, traditional Spanish, Swedish, Turkish, and Vietnamese.

Each of the 23 root-or-tailored tables supports both padding modes and all four sensitivity combinations: 184 UCA profiles. Named tailorings are invalid with binary and simple-case algorithms. See the [utf8mb4 collation mapping](20-utf8mb4-map.md) for exact external-name resolution.

## Key generation

Generate a comparison image in caller-owned scratch storage, then encode it as a field:

```go
collator, err := schottky.NewCollator(profile)
size, err := collator.KeySize(text)
scratch := make([]byte, 0, size)
comparisonKey, err := collator.Key(scratch, text)
builder.CollationKey(comparisonKey, order)
```

`Key` and `TotalKey` append only within supplied capacity. They are atomic and allocation-free when capacity is sufficient. The comparison image is not reversible text.

## Padding

`SpacePadding` ignores trailing U+0020 values. `NoPadding` keeps them significant, so `\"a\" < \"a \"`. Padding is collation behavior, not SQL storage padding; apply fixed-width type rules before key generation.

## Equality and total order

`Key` preserves collation equality. Distinct strings can produce equal keys at insensitive levels.

`TotalKey` appends an escaped copy of the original valid UTF-8 bytes after the comparison image. Use it only when an index requires deterministic bytewise ordering after collation equality. The tie-break uses original bytes, never normalized text.

## Versioning

Persist:

- `UnicodeCollationVersion`;
- `CollationProfileVersion`;
- algorithm and tailoring;
- accent and case sensitivity;
- padding;
- `Key` versus `TotalKey`;
- any outer direction and null placement.

A change to any item requires a new keyspace and a rebuild. Never compare keys from different profiles or Unicode data versions.

`Builder.CollationKey` remains available for independently produced comparison images. Its producer, version, options, and tie policy become schema metadata.
