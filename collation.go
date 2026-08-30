package schottky

import (
	"unicode/utf8"

	"gosuda.org/schottky/internal/uca"
)

// UnicodeCollationVersion identifies the immutable Unicode data used by Collator.
const UnicodeCollationVersion = uca.Version

// CollationProfileVersion identifies immutable root, tailoring, and simple-case behavior.
const CollationProfileVersion = "uca14-profile-1"

// CollationAlgorithm selects the comparison algorithm.
type CollationAlgorithm uint8

const (
	UCACollation CollationAlgorithm = iota
	BinaryCollation
	SimpleCaseCollation
)

// CollationPadding selects trailing U+0020 behavior.
type CollationPadding uint8

const (
	NoPadding CollationPadding = iota
	SpacePadding
)

// CollationTailoring selects a Unicode 14 root or language tailoring.
type CollationTailoring uint8

const (
	TailoringRoot CollationTailoring = iota
	TailoringCroatian
	TailoringCzech
	TailoringDanish
	TailoringEsperanto
	TailoringEstonian
	TailoringGermanPhonebook
	TailoringHungarian
	TailoringIcelandic
	TailoringLatvian
	TailoringLithuanian
	TailoringPersian
	TailoringPolish
	TailoringRoman
	TailoringRomanian
	TailoringSinhala
	TailoringSlovak
	TailoringSlovenian
	TailoringSpanish
	TailoringSpanishTraditional
	TailoringSwedish
	TailoringTurkish
	TailoringVietnamese
)

var ucaTailorings = [...]uca.Tailoring{
	TailoringRoot:               uca.Root,
	TailoringCroatian:           uca.Croatian,
	TailoringCzech:              uca.Czech,
	TailoringDanish:             uca.Danish,
	TailoringEsperanto:          uca.Esperanto,
	TailoringEstonian:           uca.Estonian,
	TailoringGermanPhonebook:    uca.GermanPhonebook,
	TailoringHungarian:          uca.Hungarian,
	TailoringIcelandic:          uca.Icelandic,
	TailoringLatvian:            uca.Latvian,
	TailoringLithuanian:         uca.Lithuanian,
	TailoringPersian:            uca.Persian,
	TailoringPolish:             uca.Polish,
	TailoringRoman:              uca.Roman,
	TailoringRomanian:           uca.Romanian,
	TailoringSinhala:            uca.Sinhala,
	TailoringSlovak:             uca.Slovak,
	TailoringSlovenian:          uca.Slovenian,
	TailoringSpanish:            uca.Spanish,
	TailoringSpanishTraditional: uca.SpanishTraditional,
	TailoringSwedish:            uca.Swedish,
	TailoringTurkish:            uca.Turkish,
	TailoringVietnamese:         uca.Vietnamese,
}

// CollationProfile is the complete vendor-neutral comparison contract.
type CollationProfile struct {
	Algorithm       CollationAlgorithm
	Tailoring       CollationTailoring
	Padding         CollationPadding
	AccentSensitive bool
	CaseSensitive   bool
}

// Collator produces immutable comparison keys for one profile.
type Collator struct {
	profile CollationProfile
}

// NewCollator validates and binds a collation profile.
func NewCollator(profile CollationProfile) (Collator, error) {
	if !profile.valid() {
		return Collator{}, ErrInvalidValue
	}
	return Collator{profile: profile}, nil
}

// Profile returns the bound comparison profile.
func (c Collator) Profile() CollationProfile {
	return c.profile
}

// KeySize returns the exact comparison-key size for value.
func (c Collator) KeySize(value string) (int, error) {
	value, err := c.prepare(value)
	if err != nil {
		return 0, err
	}
	switch c.profile.Algorithm {
	case BinaryCollation:
		size, ok := escapedStringSize(value)
		if !ok {
			return 0, ErrInvalidValue
		}
		return size, nil
	case SimpleCaseCollation:
		return simpleCaseKeySize(value)
	}
	size, ok := uca.KeySize(value, c.levels(), c.ucaTailoring())
	if !ok {
		return 0, ErrInvalidValue
	}
	return size, nil
}

// Key appends a comparison key to caller-owned capacity.
func (c Collator) Key(dst []byte, value string) ([]byte, error) {
	value, err := c.prepare(value)
	if err != nil {
		return dst, err
	}
	switch c.profile.Algorithm {
	case BinaryCollation:
		return appendEscapedString(dst, value)
	case SimpleCaseCollation:
		return appendSimpleCaseKey(dst, value)
	}
	key, ok := uca.AppendKey(dst, value, c.levels(), c.ucaTailoring())
	if !ok {
		if size, valid := uca.KeySize(value, c.levels(), c.ucaTailoring()); valid && size > cap(dst)-len(dst) {
			return dst, ErrShortBuffer
		}
		return dst, ErrInvalidValue
	}
	return key, nil
}

// TotalKeySize returns the exact size of a comparison key with a raw UTF-8 tie-break.
func (c Collator) TotalKeySize(value string) (int, error) {
	comparisonSize, err := c.KeySize(value)
	if err != nil {
		return 0, err
	}
	tieSize, ok := escapedStringSize(value)
	if !ok || comparisonSize > int(^uint(0)>>1)-tieSize {
		return 0, ErrInvalidValue
	}
	return comparisonSize + tieSize, nil
}

// TotalKey appends a comparison key followed by a raw UTF-8 tie-break.
func (c Collator) TotalKey(dst []byte, value string) ([]byte, error) {
	totalSize, err := c.TotalKeySize(value)
	if err != nil {
		return dst, err
	}
	if totalSize > cap(dst)-len(dst) {
		return dst, ErrShortBuffer
	}
	start := len(dst)
	key, err := c.Key(dst, value)
	if err != nil {
		return dst, err
	}
	key, err = appendEscapedString(key, value)
	if err != nil {
		return dst[:start], err
	}
	return key, nil
}

func (p CollationProfile) valid() bool {
	if p.Algorithm > SimpleCaseCollation ||
		p.Tailoring > TailoringVietnamese ||
		p.Padding > SpacePadding {
		return false
	}
	switch p.Algorithm {
	case UCACollation:
		return true
	case BinaryCollation:
		return p.Tailoring == TailoringRoot && !p.AccentSensitive && !p.CaseSensitive
	case SimpleCaseCollation:
		return p.Tailoring == TailoringRoot &&
			p.Padding == SpacePadding &&
			p.AccentSensitive &&
			!p.CaseSensitive
	default:
		return false
	}
}

func (c Collator) prepare(value string) (string, error) {
	if !c.profile.valid() || !utf8.ValidString(value) {
		return "", ErrInvalidValue
	}
	if c.profile.Padding == SpacePadding {
		for len(value) != 0 && value[len(value)-1] == ' ' {
			value = value[:len(value)-1]
		}
	}
	return value, nil
}

func (c Collator) ucaTailoring() uca.Tailoring {
	return ucaTailorings[c.profile.Tailoring]
}

func (c Collator) levels() uca.Levels {
	levels := uca.Primary
	if c.profile.AccentSensitive {
		levels |= uca.Secondary
	}
	if c.profile.CaseSensitive {
		levels |= uca.Tertiary
	}
	return levels
}

func appendEscapedString(dst []byte, value string) ([]byte, error) {
	size, ok := escapedStringSize(value)
	if !ok {
		return dst, ErrInvalidValue
	}
	if size > cap(dst)-len(dst) {
		return dst, ErrShortBuffer
	}
	start := len(dst)
	dst = dst[:start+size]
	out := start
	for i := 0; i < len(value); i++ {
		valueByte := value[i]
		if valueByte == 0 {
			dst[out] = 0
			dst[out+1] = 0xff
			out += 2
			continue
		}
		dst[out] = valueByte
		out++
	}
	dst[out] = 0
	dst[out+1] = 0
	return dst, nil
}

func simpleCaseKeySize(value string) (int, error) {
	runeCount := utf8.RuneCountInString(value)
	maxInt := int(^uint(0) >> 1)
	if runeCount > (maxInt-3)/3 {
		return 0, ErrInvalidValue
	}
	return runeCount*3 + 3, nil
}

func appendSimpleCaseKey(dst []byte, value string) ([]byte, error) {
	size, err := simpleCaseKeySize(value)
	if err != nil {
		return dst, err
	}
	if size > cap(dst)-len(dst) {
		return dst, ErrShortBuffer
	}
	start := len(dst)
	dst = dst[:start+size]
	out := start
	for _, valueRune := range value {
		ordered := uint32(uca.SimpleUpper(valueRune)) + 1
		dst[out] = byte(ordered >> 16)
		dst[out+1] = byte(ordered >> 8)
		dst[out+2] = byte(ordered)
		out += 3
	}
	clear(dst[out:])
	return dst, nil
}
