# utf8mb4 collation mapping

This integration map defines MariaDB Style for the supported utf8mb4 inventory. The public Go identifiers remain vendor-neutral.

## Scope

The pinned server reports 262 utf8mb4 collation names:

| Family | Names | Schottky behavior |
| --- | ---: | --- |
| generated UCA 14 | 184 | supported by `UCACollation` |
| compatibility aliases | 44 | mapped to an existing no-pad UCA or binary profile |
| binary | 2 | supported by `BinaryCollation` |
| Unicode-14 simple-case | 1 | supported by `SimpleCaseCollation` |
| legacy | 31 | excluded |

The supported scope is 231 names with 187 distinct behaviors. Alias names do not duplicate tables.

Use this inventory query to detect a server mismatch:

```sql
SELECT c.COLLATION_NAME, c.ID, c.IS_DEFAULT, c.SORTLEN
FROM information_schema.COLLATIONS AS c
JOIN information_schema.COLLATION_CHARACTER_SET_APPLICABILITY AS a
  ON a.COLLATION_NAME = c.COLLATION_NAME
WHERE a.CHARACTER_SET_NAME = 'utf8mb4'
ORDER BY c.COLLATION_NAME, c.ID;
```

A count other than 262 is a different target inventory and requires a mapping review.

## Generated UCA names

Names follow:

```text
uca1400[_<tailoring>][_nopad]_{ai|as}_{ci|cs}
```

Map suffixes to `CollationProfile`:

| Name component | Option |
| --- | --- |
| no `_nopad` | `Padding: SpacePadding` |
| `_nopad` | `Padding: NoPadding` |
| `_ai_` | `AccentSensitive: false` |
| `_as_` | `AccentSensitive: true` |
| `_ci` | `CaseSensitive: false` |
| `_cs` | `CaseSensitive: true` |

All use `Algorithm: UCACollation` and Unicode data version `14.0.0`.

| External tailoring | Go tailoring |
| --- | --- |
| none | `TailoringRoot` |
| `croatian` | `TailoringCroatian` |
| `czech` | `TailoringCzech` |
| `danish` | `TailoringDanish` |
| `esperanto` | `TailoringEsperanto` |
| `estonian` | `TailoringEstonian` |
| `german2` | `TailoringGermanPhonebook` |
| `hungarian` | `TailoringHungarian` |
| `icelandic` | `TailoringIcelandic` |
| `latvian` | `TailoringLatvian` |
| `lithuanian` | `TailoringLithuanian` |
| `persian` | `TailoringPersian` |
| `polish` | `TailoringPolish` |
| `roman` | `TailoringRoman` |
| `romanian` | `TailoringRomanian` |
| `sinhala` | `TailoringSinhala` |
| `slovak` | `TailoringSlovak` |
| `slovenian` | `TailoringSlovenian` |
| `spanish` | `TailoringSpanish` |
| `spanish2` | `TailoringSpanishTraditional` |
| `swedish` | `TailoringSwedish` |
| `turkish` | `TailoringTurkish` |
| `vietnamese` | `TailoringVietnamese` |

## Compatibility aliases

The `utf8mb4_0900_*` names in this target are aliases to Unicode-14 behavior, not a request for Unicode 9 data.

- `utf8mb4_0900_ai_ci`, `utf8mb4_0900_as_ci`, and `utf8mb4_0900_as_cs` map to the root no-pad UCA profile with the corresponding sensitivity flags.
- `utf8mb4_0900_bin` maps to `BinaryCollation` with `NoPadding`.
- Each locale stem below has exactly two names: `utf8mb4_<stem>_0900_ai_ci` maps to no-pad accent-insensitive/case-insensitive UCA; `utf8mb4_<stem>_0900_as_cs` maps to no-pad accent-sensitive/case-sensitive UCA.

| Locale stem | Tailoring |
| --- | --- |
| `de_pb` | `TailoringGermanPhonebook` |
| `is` | `TailoringIcelandic` |
| `lv` | `TailoringLatvian` |
| `ro` | `TailoringRomanian` |
| `sl` | `TailoringSlovenian` |
| `pl` | `TailoringPolish` |
| `et` | `TailoringEstonian` |
| `es` | `TailoringSpanish` |
| `sv` | `TailoringSwedish` |
| `tr` | `TailoringTurkish` |
| `cs` | `TailoringCzech` |
| `da` | `TailoringDanish` |
| `lt` | `TailoringLithuanian` |
| `sk` | `TailoringSlovak` |
| `es_trad` | `TailoringSpanishTraditional` |
| `la` | `TailoringRoman` |
| `eo` | `TailoringEsperanto` |
| `hu` | `TailoringHungarian` |
| `hr` | `TailoringCroatian` |
| `vi` | `TailoringVietnamese` |

## Other modern profiles

| External name | Profile |
| --- | --- |
| `utf8mb4_bin` | `BinaryCollation`, `SpacePadding` |
| `utf8mb4_nopad_bin` | `BinaryCollation`, `NoPadding` |
| `utf8mb4_general1400_as_ci` | `SimpleCaseCollation`, `SpacePadding`, accent-sensitive, case-insensitive |

`SimpleCaseCollation` compares Unicode-14 simple-uppercase scalar sequences. It does not perform UCA expansions or contractions: `é` differs from `e` plus U+0301, and `ß` differs from `ss`.

## Excluded legacy names

The clean-room utf8mb4 scope excludes:

```text
utf8mb4_croatian_ci
utf8mb4_croatian_mysql561_ci
utf8mb4_czech_ci
utf8mb4_danish_ci
utf8mb4_esperanto_ci
utf8mb4_estonian_ci
utf8mb4_general_ci
utf8mb4_general_nopad_ci
utf8mb4_german2_ci
utf8mb4_hungarian_ci
utf8mb4_icelandic_ci
utf8mb4_latvian_ci
utf8mb4_lithuanian_ci
utf8mb4_myanmar_ci
utf8mb4_persian_ci
utf8mb4_polish_ci
utf8mb4_roman_ci
utf8mb4_romanian_ci
utf8mb4_sinhala_ci
utf8mb4_slovak_ci
utf8mb4_slovenian_ci
utf8mb4_spanish_ci
utf8mb4_spanish2_ci
utf8mb4_swedish_ci
utf8mb4_thai_520_w2
utf8mb4_turkish_ci
utf8mb4_unicode_ci
utf8mb4_unicode_nopad_ci
utf8mb4_unicode_520_ci
utf8mb4_unicode_520_nopad_ci
utf8mb4_vietnamese_ci
```

Selecting one of these names is a configuration error; do not silently substitute a modern profile.

## Key policy

Use `Collator.Key` to preserve SQL collation equality. Use `Collator.TotalKey` only for a separate deterministic application order; the server itself does not add that raw-byte tie-break to these collations.

Persist the style label, external name, resolved `CollationProfile`, `UnicodeCollationVersion`, `CollationProfileVersion`, and key policy. Rebuild keys when any value changes.
