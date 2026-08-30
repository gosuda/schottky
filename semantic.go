package schottky

import (
	"math"
	"math/bits"

	"gosuda.org/schottky/internal/byteops"
)

const (
	microsPerDay       int64 = 86_400_000_000
	maxUTCOffsetSecond int32 = 15*60*60 + 59*60 + 59
	dateFiniteMin      int32 = -2_451_545
	dateFiniteEnd      int32 = 2_145_031_949
	timestampFiniteMin int64 = -211_813_488_000_000_000
	timestampFiniteEnd int64 = 9_223_371_331_200_000_000
)

const (
	DateNegativeInfinity      int32 = math.MinInt32
	DatePositiveInfinity      int32 = math.MaxInt32
	TimestampNegativeInfinity int64 = math.MinInt64
	TimestampPositiveInfinity int64 = math.MaxInt64
)

// ZonedTime stores local microseconds and a UTC offset measured eastward.
type ZonedTime struct {
	Microseconds     int64
	UTCOffsetSeconds int32
}

// Date appends signed days since 2000-01-01 or a date infinity sentinel.
func (b *Builder) Date(days int32, order Order) {
	if !validDate(days) {
		b.fail(ErrInvalidValue)
		return
	}
	b.Int32(days, order)
}

// Time appends microseconds since midnight, including the end-of-day sentinel.
func (b *Builder) Time(microseconds int64, order Order) {
	if microseconds < 0 || microseconds > microsPerDay {
		b.fail(ErrInvalidValue)
		return
	}
	b.Int64(microseconds, order)
}

// ZonedTime appends a time with a numeric UTC offset.
func (b *Builder) ZonedTime(value ZonedTime, order Order) {
	if !validZonedTime(value) {
		b.fail(ErrInvalidValue)
		return
	}
	westOffset := -value.UTCOffsetSeconds
	utcMicroseconds := value.Microseconds + int64(westOffset)*1_000_000
	payload, ok := b.begin(order, 12)
	if !ok {
		return
	}
	putOrdered(payload[:8], uint64(utcMicroseconds)^0x8000000000000000)
	putOrdered(payload[8:], uint64(uint32(westOffset)^0x80000000))
	if order.descending() {
		byteops.Invert(payload)
	}
}

// Timestamp appends microseconds since 2000-01-01 or a timestamp infinity sentinel.
func (b *Builder) Timestamp(microseconds int64, order Order) {
	if !validTimestamp(microseconds) {
		b.fail(ErrInvalidValue)
		return
	}
	b.Int64(microseconds, order)
}

// Duration appends signed elapsed microseconds.
func (b *Builder) Duration(microseconds int64, order Order) {
	b.Int64(microseconds, order)
}

// IntervalOrderValue returns the exact 30-day-month comparison scalar.
func IntervalOrderValue(months, days int32, microseconds int64) Int128 {
	switch {
	case months == math.MinInt32 && days == math.MinInt32 && microseconds == math.MinInt64:
		return Int128{High: math.MinInt64}
	case months == math.MaxInt32 && days == math.MaxInt32 && microseconds == math.MaxInt64:
		return Int128{High: math.MaxInt64, Low: math.MaxUint64}
	}
	dayCount := int64(months)*30 + int64(days)
	span := multiplySigned(dayCount, uint64(microsPerDay))
	return addSigned(span, microseconds)
}

// Enum appends an immutable enum rank.
func (b *Builder) Enum(rank uint32, order Order) {
	b.Uint32(rank, order)
}

// LSN appends an unsigned 64-bit log position.
func (b *Builder) LSN(value uint64, order Order) {
	b.Uint64(value, order)
}

func (d *Decoder) Date(order Order) (int32, Presence) {
	value, presence := d.Int32(order)
	if d.err != nil || presence != Present {
		return value, presence
	}
	if !validDate(value) {
		d.err = ErrMalformedKey
		return 0, Present
	}
	return value, Present
}

func (d *Decoder) Time(order Order) (int64, Presence) {
	value, presence := d.Int64(order)
	if d.err != nil || presence != Present {
		return value, presence
	}
	if value < 0 || value > microsPerDay {
		d.err = ErrMalformedKey
		return 0, Present
	}
	return value, Present
}

func (d *Decoder) ZonedTime(order Order) (ZonedTime, Presence) {
	var raw [12]byte
	presence, ok := d.fixed(order, len(raw), raw[:])
	if !ok || presence == Null {
		return ZonedTime{}, presence
	}
	utcMicroseconds := int64(readOrdered(raw[:8]) ^ 0x8000000000000000)
	westOffset := int32(uint32(readOrdered(raw[8:])) ^ 0x80000000)
	value := ZonedTime{
		Microseconds:     utcMicroseconds - int64(westOffset)*1_000_000,
		UTCOffsetSeconds: -westOffset,
	}
	if !validZonedTime(value) {
		d.err = ErrMalformedKey
		return ZonedTime{}, Present
	}
	return value, Present
}

func (d *Decoder) Timestamp(order Order) (int64, Presence) {
	value, presence := d.Int64(order)
	if d.err != nil || presence != Present {
		return value, presence
	}
	if !validTimestamp(value) {
		d.err = ErrMalformedKey
		return 0, Present
	}
	return value, Present
}

func (d *Decoder) Duration(order Order) (int64, Presence) {
	return d.Int64(order)
}

func (d *Decoder) Enum(order Order) (uint32, Presence) {
	return d.Uint32(order)
}

func (d *Decoder) LSN(order Order) (uint64, Presence) {
	return d.Uint64(order)
}

func validDate(days int32) bool {
	return days == DateNegativeInfinity ||
		days == DatePositiveInfinity ||
		days >= dateFiniteMin && days < dateFiniteEnd
}

func validTimestamp(microseconds int64) bool {
	return microseconds == TimestampNegativeInfinity ||
		microseconds == TimestampPositiveInfinity ||
		microseconds >= timestampFiniteMin && microseconds < timestampFiniteEnd
}

func validZonedTime(value ZonedTime) bool {
	return value.Microseconds >= 0 &&
		value.Microseconds <= microsPerDay &&
		value.UTCOffsetSeconds >= -maxUTCOffsetSecond &&
		value.UTCOffsetSeconds <= maxUTCOffsetSecond
}

func multiplySigned(value int64, factor uint64) Int128 {
	negative := value < 0
	magnitude := uint64(value)
	if negative {
		magnitude = ^magnitude + 1
	}
	high, low := bits.Mul64(magnitude, factor)
	if negative {
		high = ^high
		low = ^low + 1
		if low == 0 {
			high++
		}
	}
	return Int128{High: int64(high), Low: low}
}

func addSigned(value Int128, addend int64) Int128 {
	low, carry := bits.Add64(value.Low, uint64(addend), 0)
	highAddend := uint64(0)
	if addend < 0 {
		highAddend = math.MaxUint64
	}
	high, _ := bits.Add64(uint64(value.High), highAddend, carry)
	return Int128{High: int64(high), Low: low}
}

func putOrdered(dst []byte, value uint64) {
	for i := len(dst) - 1; i >= 0; i-- {
		dst[i] = byte(value)
		value >>= 8
	}
}

func readOrdered(src []byte) uint64 {
	var value uint64
	for _, valueByte := range src {
		value = value<<8 | uint64(valueByte)
	}
	return value
}
