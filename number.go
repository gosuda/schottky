package schottky

import (
	"math"

	"gosuda.org/schottky/internal/byteops"
)

// Int128 stores a signed two's-complement 128-bit integer.
type Int128 struct {
	High int64
	Low  uint64
}

// Bool appends a Boolean field.
func (b *Builder) Bool(value bool, order Order) {
	payload, ok := b.begin(order, 1)
	if !ok {
		return
	}
	if value {
		payload[0] = 1
	}
	if order.descending() {
		payload[0] = ^payload[0]
	}
}

func (b *Builder) Int8(value int8, order Order) {
	b.orderedUint(uint64(uint8(value)^0x80), 1, order)
}

func (b *Builder) Int16(value int16, order Order) {
	b.orderedUint(uint64(uint16(value)^0x8000), 2, order)
}

func (b *Builder) Int32(value int32, order Order) {
	b.orderedUint(uint64(uint32(value)^0x80000000), 4, order)
}

func (b *Builder) Int64(value int64, order Order) {
	b.orderedUint(uint64(value)^0x8000000000000000, 8, order)
}

// Int128 appends a signed 128-bit integer.
func (b *Builder) Int128(value Int128, order Order) {
	payload, ok := b.begin(order, 16)
	if !ok {
		return
	}
	high := uint64(value.High) ^ 0x8000000000000000
	for i := 7; i >= 0; i-- {
		payload[i] = byte(high)
		high >>= 8
	}
	low := value.Low
	for i := 15; i >= 8; i-- {
		payload[i] = byte(low)
		low >>= 8
	}
	if order.descending() {
		byteops.Invert(payload)
	}
}

func (b *Builder) Uint8(value uint8, order Order) {
	b.orderedUint(uint64(value), 1, order)
}

func (b *Builder) Uint16(value uint16, order Order) {
	b.orderedUint(uint64(value), 2, order)
}

func (b *Builder) Uint32(value uint32, order Order) {
	b.orderedUint(uint64(value), 4, order)
}

func (b *Builder) Uint64(value uint64, order Order) {
	b.orderedUint(value, 8, order)
}

// Float32 appends a database-total-ordered IEEE binary32 field.
func (b *Builder) Float32(value float32, order Order) {
	bits := math.Float32bits(value)
	var ordered uint32
	switch {
	case math.IsNaN(float64(value)):
		ordered = math.MaxUint32
	case value == 0:
		ordered = 0x80000000
	case bits&0x80000000 != 0:
		ordered = ^bits
	default:
		ordered = bits ^ 0x80000000
	}
	b.orderedUint(uint64(ordered), 4, order)
}

// Float64 appends a database-total-ordered IEEE binary64 field.
func (b *Builder) Float64(value float64, order Order) {
	bits := math.Float64bits(value)
	var ordered uint64
	switch {
	case math.IsNaN(value):
		ordered = math.MaxUint64
	case value == 0:
		ordered = 0x8000000000000000
	case bits&0x8000000000000000 != 0:
		ordered = ^bits
	default:
		ordered = bits ^ 0x8000000000000000
	}
	b.orderedUint(ordered, 8, order)
}

func (b *Builder) orderedUint(value uint64, size int, order Order) {
	payload, ok := b.begin(order, size)
	if !ok {
		return
	}
	for i := size - 1; i >= 0; i-- {
		payload[i] = byte(value)
		value >>= 8
	}
	if order.descending() {
		byteops.Invert(payload)
	}
}

func (d *Decoder) Bool(order Order) (bool, Presence) {
	value, presence, ok := d.uint(order, 1)
	if !ok || presence == Null {
		return false, presence
	}
	if value > 1 {
		d.err = ErrMalformedKey
		return false, Present
	}
	return value == 1, Present
}

func (d *Decoder) Int8(order Order) (int8, Presence) {
	value, presence, ok := d.uint(order, 1)
	if !ok || presence == Null {
		return 0, presence
	}
	return int8(uint8(value) ^ 0x80), Present
}

func (d *Decoder) Int16(order Order) (int16, Presence) {
	value, presence, ok := d.uint(order, 2)
	if !ok || presence == Null {
		return 0, presence
	}
	return int16(uint16(value) ^ 0x8000), Present
}

func (d *Decoder) Int32(order Order) (int32, Presence) {
	value, presence, ok := d.uint(order, 4)
	if !ok || presence == Null {
		return 0, presence
	}
	return int32(uint32(value) ^ 0x80000000), Present
}

func (d *Decoder) Int64(order Order) (int64, Presence) {
	value, presence, ok := d.uint(order, 8)
	if !ok || presence == Null {
		return 0, presence
	}
	return int64(value ^ 0x8000000000000000), Present
}

func (d *Decoder) Int128(order Order) (Int128, Presence) {
	var raw [16]byte
	presence, ok := d.fixed(order, len(raw), raw[:])
	if !ok || presence == Null {
		return Int128{}, presence
	}
	var high uint64
	for _, valueByte := range raw[:8] {
		high = high<<8 | uint64(valueByte)
	}
	var low uint64
	for _, valueByte := range raw[8:] {
		low = low<<8 | uint64(valueByte)
	}
	return Int128{High: int64(high ^ 0x8000000000000000), Low: low}, Present
}

func (d *Decoder) Uint8(order Order) (uint8, Presence) {
	value, presence, ok := d.uint(order, 1)
	return uint8(value), decodedPresence(presence, ok)
}

func (d *Decoder) Uint16(order Order) (uint16, Presence) {
	value, presence, ok := d.uint(order, 2)
	return uint16(value), decodedPresence(presence, ok)
}

func (d *Decoder) Uint32(order Order) (uint32, Presence) {
	value, presence, ok := d.uint(order, 4)
	return uint32(value), decodedPresence(presence, ok)
}

func (d *Decoder) Uint64(order Order) (uint64, Presence) {
	value, presence, ok := d.uint(order, 8)
	return value, decodedPresence(presence, ok)
}

func (d *Decoder) Float32(order Order) (float32, Presence) {
	value, presence, ok := d.uint(order, 4)
	if !ok || presence == Null {
		return 0, presence
	}
	ordered := uint32(value)
	if ordered == math.MaxUint32 {
		return float32(math.NaN()), Present
	}
	bits := ^ordered
	if ordered&0x80000000 != 0 {
		bits = ordered ^ 0x80000000
	}
	decoded := math.Float32frombits(bits)
	if math.IsNaN(float64(decoded)) || bits == 0x80000000 {
		d.err = ErrMalformedKey
		return 0, Present
	}
	return decoded, Present
}

func (d *Decoder) Float64(order Order) (float64, Presence) {
	ordered, presence, ok := d.uint(order, 8)
	if !ok || presence == Null {
		return 0, presence
	}
	if ordered == math.MaxUint64 {
		return math.NaN(), Present
	}
	bits := ^ordered
	if ordered&0x8000000000000000 != 0 {
		bits = ordered ^ 0x8000000000000000
	}
	decoded := math.Float64frombits(bits)
	if math.IsNaN(decoded) || bits == 0x8000000000000000 {
		d.err = ErrMalformedKey
		return 0, Present
	}
	return decoded, Present
}

func (d *Decoder) uint(order Order, size int) (uint64, Presence, bool) {
	var raw [8]byte
	presence, ok := d.fixed(order, size, raw[:size])
	if !ok || presence == Null {
		return 0, presence, ok
	}
	var value uint64
	for _, valueByte := range raw[:size] {
		value = value<<8 | uint64(valueByte)
	}
	return value, Present, true
}

func decodedPresence(presence Presence, ok bool) Presence {
	if !ok {
		return Present
	}
	return presence
}
