package schottky

import "gosuda.org/schottky/internal/byteops"

// UUID appends 16 canonical network-order UUID bytes.
func (b *Builder) UUID(value [16]byte, order Order) {
	b.fixedBytes(value[:], order)
}

// MAC48 appends six canonical address bytes.
func (b *Builder) MAC48(value [6]byte, order Order) {
	b.fixedBytes(value[:], order)
}

// MAC64 appends eight canonical address bytes.
func (b *Builder) MAC64(value [8]byte, order Order) {
	b.fixedBytes(value[:], order)
}

func (b *Builder) fixedBytes(value []byte, order Order) {
	payload, ok := b.begin(order, len(value))
	if !ok {
		return
	}
	copy(payload, value)
	if order.descending() {
		byteops.Invert(payload)
	}
}

func (d *Decoder) UUID(order Order) ([16]byte, Presence) {
	var value [16]byte
	presence, _ := d.fixed(order, len(value), value[:])
	return value, presence
}

func (d *Decoder) MAC48(order Order) ([6]byte, Presence) {
	var value [6]byte
	presence, _ := d.fixed(order, len(value), value[:])
	return value, presence
}

func (d *Decoder) MAC64(order Order) ([8]byte, Presence) {
	var value [8]byte
	presence, _ := d.fixed(order, len(value), value[:])
	return value, presence
}
