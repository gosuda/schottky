package schottky

import (
	"strconv"

	"gosuda.org/schottky/internal/byteops"
)

const (
	decimalNegativeInfinity byte = iota
	decimalNegativeFinite
	decimalZero
	decimalPositiveFinite
	decimalPositiveInfinity
	decimalNaN
)

type decimalParts struct {
	class         byte
	mantissaStart int
	mantissaEnd   int
	firstDigit    int
	lastDigit     int
	digitCount    int
	adjusted      int32
}

// Decimal appends a normalized arbitrary-precision decimal parsed from value.
func (b *Builder) Decimal(value string, order Order) {
	parts, ok := parseDecimal(value)
	if !ok {
		b.fail(ErrInvalidValue)
		return
	}

	payloadSize := 1
	if parts.class == decimalNegativeFinite || parts.class == decimalPositiveFinite {
		payloadSize += 4 + parts.digitCount + 1
	}
	payload, ok := b.begin(order, payloadSize)
	if !ok {
		return
	}
	payload[0] = parts.class
	if payloadSize > 1 {
		orderedExponent := uint32(parts.adjusted) ^ 0x80000000
		payload[1] = byte(orderedExponent >> 24)
		payload[2] = byte(orderedExponent >> 16)
		payload[3] = byte(orderedExponent >> 8)
		payload[4] = byte(orderedExponent)

		ordinal := 0
		out := 5
		for i := parts.mantissaStart; i < parts.mantissaEnd; i++ {
			if value[i] == '.' {
				continue
			}
			if ordinal >= parts.firstDigit && ordinal <= parts.lastDigit {
				payload[out] = value[i] - '0' + 1
				out++
			}
			ordinal++
		}
		payload[out] = 0
		if parts.class == decimalNegativeFinite {
			byteops.Invert(payload[1:])
		}
	}
	if order.descending() {
		byteops.Invert(payload)
	}
}

// Decimal decodes a canonical coefficient-and-exponent string into dst.
func (d *Decoder) Decimal(dst []byte, order Order) ([]byte, Presence) {
	presence, ok := d.presence(order)
	if !ok || presence == Null {
		return dst, presence
	}
	if d.off == len(d.src) {
		d.err = ErrMalformedKey
		return dst, Present
	}

	class := d.decimalByte(d.off, order, false)
	switch class {
	case decimalNegativeInfinity:
		return d.appendDecimalLiteral(dst, "-Infinity", 1)
	case decimalZero:
		return d.appendDecimalLiteral(dst, "0", 1)
	case decimalPositiveInfinity:
		return d.appendDecimalLiteral(dst, "Infinity", 1)
	case decimalNaN:
		return d.appendDecimalLiteral(dst, "NaN", 1)
	case decimalNegativeFinite, decimalPositiveFinite:
		return d.decodeFiniteDecimal(dst, order, class)
	default:
		d.err = ErrMalformedKey
		return dst, Present
	}
}

func (d *Decoder) decodeFiniteDecimal(dst []byte, order Order, class byte) ([]byte, Presence) {
	negative := class == decimalNegativeFinite
	if len(d.src)-d.off < 6 {
		d.err = ErrMalformedKey
		return dst, Present
	}

	var orderedExponent uint32
	for i := 1; i <= 4; i++ {
		orderedExponent = orderedExponent<<8 | uint32(d.decimalByte(d.off+i, order, negative))
	}
	adjusted := int32(orderedExponent ^ 0x80000000)

	digitsStart := d.off + 5
	digitsEnd := digitsStart
	for ; digitsEnd < len(d.src); digitsEnd++ {
		valueByte := d.decimalByte(digitsEnd, order, negative)
		if valueByte == 0 {
			break
		}
		if valueByte > 10 {
			d.err = ErrMalformedKey
			return dst, Present
		}
	}
	if digitsEnd == len(d.src) || digitsEnd == digitsStart {
		d.err = ErrMalformedKey
		return dst, Present
	}
	if d.decimalByte(digitsStart, order, negative) == 1 || d.decimalByte(digitsEnd-1, order, negative) == 1 {
		d.err = ErrMalformedKey
		return dst, Present
	}

	digitCount := digitsEnd - digitsStart
	coefficientExponent := int64(adjusted) - int64(digitCount-1)
	needed := digitCount + 1 + signedDecimalLen(coefficientExponent)
	if negative {
		needed++
	}
	if needed > cap(dst)-len(dst) {
		d.err = ErrShortBuffer
		return dst, Present
	}

	if negative {
		dst = append(dst, '-')
	}
	for i := digitsStart; i < digitsEnd; i++ {
		dst = append(dst, d.decimalByte(i, order, negative)-1+'0')
	}
	dst = append(dst, 'e')
	dst = strconv.AppendInt(dst, coefficientExponent, 10)
	d.off = digitsEnd + 1
	return dst, Present
}

func (d *Decoder) decimalByte(offset int, order Order, invertMagnitude bool) byte {
	value := d.src[offset]
	if order.descending() {
		value = ^value
	}
	if invertMagnitude {
		value = ^value
	}
	return value
}

func (d *Decoder) appendDecimalLiteral(dst []byte, literal string, consumed int) ([]byte, Presence) {
	if len(literal) > cap(dst)-len(dst) {
		d.err = ErrShortBuffer
		return dst, Present
	}
	dst = append(dst, literal...)
	d.off += consumed
	return dst, Present
}

func parseDecimal(value string) (decimalParts, bool) {
	parts := decimalParts{firstDigit: -1, lastDigit: -1}
	if value == "" {
		return parts, false
	}

	start := 0
	negative := false
	if value[0] == '+' || value[0] == '-' {
		negative = value[0] == '-'
		start++
		if start == len(value) {
			return parts, false
		}
	}
	if value[start:] == "Infinity" {
		if negative {
			parts.class = decimalNegativeInfinity
		} else {
			parts.class = decimalPositiveInfinity
		}
		return parts, true
	}
	if value[start:] == "NaN" {
		parts.class = decimalNaN
		return parts, true
	}

	mantissaEnd := len(value)
	exponent := int64(0)
	for i := start; i < len(value); i++ {
		if value[i] != 'e' && value[i] != 'E' {
			continue
		}
		if mantissaEnd != len(value) {
			return parts, false
		}
		mantissaEnd = i
		var ok bool
		exponent, ok = parseDecimalExponent(value[i+1:], len(value))
		if !ok {
			return parts, false
		}
	}

	totalDigits := 0
	digitsBeforePoint := -1
	seenPoint := false
	for i := start; i < mantissaEnd; i++ {
		valueByte := value[i]
		if valueByte == '.' {
			if seenPoint {
				return parts, false
			}
			seenPoint = true
			digitsBeforePoint = totalDigits
			continue
		}
		if valueByte < '0' || valueByte > '9' {
			return parts, false
		}
		if valueByte != '0' {
			if parts.firstDigit < 0 {
				parts.firstDigit = totalDigits
			}
			parts.lastDigit = totalDigits
		}
		totalDigits++
	}
	if totalDigits == 0 {
		return parts, false
	}
	if digitsBeforePoint < 0 {
		digitsBeforePoint = totalDigits
	}
	if parts.firstDigit < 0 {
		parts.class = decimalZero
		return parts, true
	}

	parts.digitCount = parts.lastDigit - parts.firstDigit + 1
	fractionDigits := totalDigits - digitsBeforePoint
	trailingZeros := totalDigits - parts.lastDigit - 1
	adjusted := exponent - int64(fractionDigits) + int64(trailingZeros+parts.digitCount-1)
	if adjusted < -1<<31 || adjusted > 1<<31-1 {
		return parts, false
	}
	parts.adjusted = int32(adjusted)
	parts.mantissaStart = start
	parts.mantissaEnd = mantissaEnd
	if negative {
		parts.class = decimalNegativeFinite
	} else {
		parts.class = decimalPositiveFinite
	}
	return parts, true
}

func parseDecimalExponent(value string, inputSize int) (int64, bool) {
	if value == "" {
		return 0, false
	}
	negative := false
	start := 0
	if value[0] == '+' || value[0] == '-' {
		negative = value[0] == '-'
		start++
		if start == len(value) {
			return 0, false
		}
	}
	limit := int64(1<<31-1) + int64(inputSize) + 1
	var exponent int64
	for i := start; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		digit := int64(value[i] - '0')
		if exponent > (limit-digit)/10 {
			return 0, false
		}
		exponent = exponent*10 + digit
	}
	if negative {
		exponent = -exponent
	}
	return exponent, true
}

func signedDecimalLen(value int64) int {
	length := 1
	if value < 0 {
		length++
		value = -value
	}
	for value >= 10 {
		value /= 10
		length++
	}
	return length
}
