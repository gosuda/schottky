// Package uca implements versioned Unicode collation tables without
// normalization.
package uca

import (
	"encoding/binary"
	"unicode/utf8"
)

//go:generate go run ./gen -output tables_generated.go
//go:generate go run ./tailorgen -input tailoring_oracle.json -output tailoring_generated.go

// Version is the Unicode Collation Algorithm version implemented by this package.
const Version = "14.0.0"

// Levels selects collation-weight levels for a key.
type Levels uint8

const (
	Primary Levels = 1 << iota
	Secondary
	Tertiary
)

const allLevels = Primary | Secondary | Tertiary

// Tailoring selects a root or language-specific ordering table.
type Tailoring uint8

const (
	Root Tailoring = iota
	Croatian
	Czech
	Danish
	Esperanto
	Estonian
	GermanPhonebook
	Hungarian
	Icelandic
	Latvian
	Lithuanian
	Persian
	Polish
	Roman
	Romanian
	Sinhala
	Slovak
	Slovenian
	Spanish
	SpanishTraditional
	Swedish
	Turkish
	Vietnamese
)

func (tailoring Tailoring) valid() bool {
	return tailoring <= Vietnamese
}

// SimpleUpper returns the Unicode 14.0.0 simple uppercase mapping for r.
// Runes without a one-scalar uppercase mapping are returned unchanged.
func SimpleUpper(r rune) rune {
	if r < 0 {
		return r
	}
	codePoint := uint32(r)
	low, high := 0, len(simpleUpperRanges)
	for low < high {
		middle := int(uint(low+high) >> 1)
		candidate := simpleUpperRanges[middle]
		if codePoint < candidate.first {
			high = middle
		} else if codePoint > candidate.last {
			low = middle + 1
		} else {
			if (codePoint-candidate.first)%candidate.stride != 0 {
				return r
			}
			return r + rune(candidate.delta)
		}
	}
	return r
}

// KeySize reports the number of bytes AppendKey would append.
func KeySize(text string, levels Levels, tailoring Tailoring) (int, bool) {
	_, _, _, size, ok := measure(text, levels, tailoring)
	return size, ok
}

// AppendKey appends a sortable Unicode collation key to dst. It does not
// modify dst when any argument is invalid or dst has insufficient capacity.
func AppendKey(dst []byte, text string, levels Levels, tailoring Tailoring) ([]byte, bool) {
	primarySize, secondarySize, _, size, ok := measure(text, levels, tailoring)
	if !ok || size > cap(dst)-len(dst) {
		return dst, false
	}

	base := len(dst)
	out := dst[:base+size]
	primaryAt := base
	secondaryAt := base + primarySize
	tertiaryAt := secondaryAt + secondarySize

	iterator := elementIterator{text: text, tailoring: tailoring, valid: true}
	for {
		primary, secondary, tertiary, more := iterator.next()
		if !more {
			break
		}
		if levels&Primary != 0 && primary != 0 {
			binary.BigEndian.PutUint16(out[primaryAt:], primary)
			primaryAt += 2
		}
		if levels&Secondary != 0 && secondary != 0 {
			binary.BigEndian.PutUint16(out[secondaryAt:], secondary)
			secondaryAt += 2
		}
		if levels&Tertiary != 0 && tertiary != 0 {
			binary.BigEndian.PutUint16(out[tertiaryAt:], tertiary)
			tertiaryAt += 2
		}
	}

	if levels&Primary != 0 {
		binary.BigEndian.PutUint16(out[primaryAt:], 0)
	}
	if levels&Secondary != 0 {
		binary.BigEndian.PutUint16(out[secondaryAt:], 0)
	}
	if levels&Tertiary != 0 {
		binary.BigEndian.PutUint16(out[tertiaryAt:], 0)
	}
	return out, true
}

func measure(text string, levels Levels, tailoring Tailoring) (primary, secondary, tertiary, total int, ok bool) {
	if levels == 0 || levels&^allLevels != 0 || !tailoring.valid() {
		return 0, 0, 0, 0, false
	}

	if levels&Primary != 0 {
		primary = 2
		total = 2
	}
	if levels&Secondary != 0 {
		secondary = 2
		if !addSize(&total, 2) {
			return 0, 0, 0, 0, false
		}
	}
	if levels&Tertiary != 0 {
		tertiary = 2
		if !addSize(&total, 2) {
			return 0, 0, 0, 0, false
		}
	}

	iterator := elementIterator{text: text, tailoring: tailoring, valid: true}
	for {
		first, second, third, more := iterator.next()
		if !more {
			break
		}
		if levels&Primary != 0 && first != 0 {
			if !addSize(&primary, 2) || !addSize(&total, 2) {
				return 0, 0, 0, 0, false
			}
		}
		if levels&Secondary != 0 && second != 0 {
			if !addSize(&secondary, 2) || !addSize(&total, 2) {
				return 0, 0, 0, 0, false
			}
		}
		if levels&Tertiary != 0 && third != 0 {
			if !addSize(&tertiary, 2) || !addSize(&total, 2) {
				return 0, 0, 0, 0, false
			}
		}
	}
	if !iterator.valid {
		return 0, 0, 0, 0, false
	}
	return primary, secondary, tertiary, total, true
}

func addSize(size *int, increment int) bool {
	maximum := int(^uint(0) >> 1)
	if *size > maximum-increment {
		return false
	}
	*size += increment
	return true
}

type elementIterator struct {
	text      string
	pos       int
	tailoring Tailoring

	weightAt        uint32
	weightEnd       uint32
	tailoringWeight bool

	implicitLead  uint16
	implicitTrail uint16
	implicitPhase uint8
	valid         bool
}

func (iterator *elementIterator) next() (uint16, uint16, uint16, bool) {
	for {
		if iterator.weightAt < iterator.weightEnd {
			at := iterator.weightAt
			iterator.weightAt += 3
			if iterator.tailoringWeight {
				return tailoringWeights[at], tailoringWeights[at+1], tailoringWeights[at+2], true
			}
			return collationWeights[at], collationWeights[at+1], collationWeights[at+2], true
		}
		if iterator.implicitPhase != 0 {
			if iterator.implicitPhase == 2 {
				iterator.implicitPhase = 1
				return iterator.implicitLead, 0x0020, 0x0002, true
			}
			iterator.implicitPhase = 0
			return iterator.implicitTrail, 0, 0, true
		}
		if iterator.pos == len(iterator.text) {
			return 0, 0, 0, false
		}

		match, valid := lookupRoot(iterator.text, iterator.pos)
		if !valid {
			iterator.valid = false
			return 0, 0, 0, false
		}
		if tailored, found := lookupTailoring(iterator.text, iterator.pos, match.codePoint, iterator.tailoring); found && tailored.width >= match.width {
			match = tailored
		}
		iterator.pos += match.width
		if match.mapping != 0 {
			iterator.setMapping(match.mapping, match.tailored)
			continue
		}

		iterator.implicitLead, iterator.implicitTrail = implicitWeights(match.codePoint)
		iterator.implicitPhase = 2
	}
}

type rootMatch struct {
	codePoint uint32
	mapping   uint32
	width     int
	tailored  bool
}

func lookupRoot(text string, pos int) (rootMatch, bool) {
	codePoint, width := utf8.DecodeRuneInString(text[pos:])
	if codePoint == utf8.RuneError && width == 1 {
		return rootMatch{}, false
	}

	match := rootMatch{codePoint: uint32(codePoint), width: width}
	if mapping, extraWidth, found := lookupContraction(text, pos+width, match.codePoint); found {
		match.mapping = mapping
		match.width += extraWidth
		return match, true
	}
	match.mapping, _ = lookupScalar(match.codePoint)
	return match, true
}

func (iterator *elementIterator) setMapping(value uint32, tailored bool) {
	count := value & mappingCountMask
	iterator.weightAt = value >> mappingCountBits
	iterator.weightEnd = iterator.weightAt + count*3
	iterator.tailoringWeight = tailored
}

func lookupTailoring(text string, pos int, first uint32, tailoring Tailoring) (rootMatch, bool) {
	if tailoring == Root {
		return rootMatch{}, false
	}
	low := int(tailoringProfileBounds[tailoring])
	high := int(tailoringProfileBounds[tailoring+1])
	if low == high {
		return rootMatch{}, false
	}

	_, firstWidth := utf8.DecodeRuneInString(text[pos:])
	if second, secondWidth, valid := decodeValidRune(text, pos+firstWidth); valid {
		if mapping, found := findTailoring(low, high, first, second); found {
			return rootMatch{
				codePoint: first,
				mapping:   mapping,
				width:     firstWidth + secondWidth,
				tailored:  true,
			}, true
		}
	}
	if mapping, found := findTailoring(low, high, first, 0); found {
		return rootMatch{
			codePoint: first,
			mapping:   mapping,
			width:     firstWidth,
			tailored:  true,
		}, true
	}
	return rootMatch{}, false
}

func findTailoring(low, high int, first, second uint32) (uint32, bool) {
	limit := high
	for low < high {
		middle := int(uint(low+high) >> 1)
		candidate := tailoringRecords[middle]
		if candidate.first < first || candidate.first == first && candidate.second < second {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low == limit {
		return 0, false
	}
	candidate := tailoringRecords[low]
	if candidate.first != first || candidate.second != second {
		return 0, false
	}
	return candidate.mapping, true
}

func lookupScalar(codePoint uint32) (uint32, bool) {
	low, high := 0, len(scalarCodePoints)
	for low < high {
		middle := int(uint(low+high) >> 1)
		candidate := scalarCodePoints[middle]
		if candidate < codePoint {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low == len(scalarCodePoints) || scalarCodePoints[low] != codePoint {
		return 0, false
	}
	return scalarMappings[low], true
}

func lookupContraction(text string, pos int, first uint32) (uint32, int, bool) {
	low := lowerContraction(first, 0, 0)
	if low == len(contractions) || contractions[low].first != first {
		return 0, 0, false
	}

	second, secondWidth, valid := decodeValidRune(text, pos)
	if !valid {
		return 0, 0, false
	}
	third, thirdWidth, thirdValid := decodeValidRune(text, pos+secondWidth)
	if thirdValid {
		if value, found := findContraction(first, second, third); found {
			return value, secondWidth + thirdWidth, true
		}
	}
	if value, found := findContraction(first, second, 0); found {
		return value, secondWidth, true
	}
	return 0, 0, false
}

func decodeValidRune(text string, pos int) (uint32, int, bool) {
	if pos >= len(text) {
		return 0, 0, false
	}
	codePoint, width := utf8.DecodeRuneInString(text[pos:])
	if codePoint == utf8.RuneError && width == 1 {
		return 0, 0, false
	}
	return uint32(codePoint), width, true
}

func findContraction(first, second, third uint32) (uint32, bool) {
	at := lowerContraction(first, second, third)
	if at == len(contractions) {
		return 0, false
	}
	entry := contractions[at]
	if entry.first != first || entry.second != second || entry.third != third {
		return 0, false
	}
	return entry.mapping, true
}

func lowerContraction(first, second, third uint32) int {
	low, high := 0, len(contractions)
	for low < high {
		middle := int(uint(low+high) >> 1)
		entry := contractions[middle]
		if entry.first < first || entry.first == first && (entry.second < second || entry.second == second && entry.third < third) {
			low = middle + 1
		} else {
			high = middle
		}
	}
	return low
}

func implicitWeights(codePoint uint32) (uint16, uint16) {
	for _, implicit := range specialImplicitRanges {
		if codePoint >= implicit.first && codePoint <= implicit.last {
			return implicit.lead, uint16(codePoint-implicit.origin) | 0x8000
		}
	}

	if isUnifiedIdeograph(codePoint) {
		base := uint32(0xFB80)
		if codePoint >= 0x4E00 && codePoint <= 0x9FFF || codePoint >= 0xF900 && codePoint <= 0xFAFF {
			base = 0xFB40
		}
		return uint16(base + codePoint>>15), uint16(codePoint&0x7FFF) | 0x8000
	}
	return uint16(0xFBC0 + codePoint>>15), uint16(codePoint&0x7FFF) | 0x8000
}

func isUnifiedIdeograph(codePoint uint32) bool {
	low, high := 0, len(unifiedIdeographRanges)
	for low < high {
		middle := int(uint(low+high) >> 1)
		candidate := unifiedIdeographRanges[middle]
		if codePoint < candidate.first {
			high = middle
		} else if codePoint > candidate.last {
			low = middle + 1
		} else {
			return true
		}
	}
	return false
}
