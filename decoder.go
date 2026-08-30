package schottky

// Decoder consumes a key according to its external schema.
type Decoder struct {
	src []byte
	off int
	err error
}

// NewDecoder borrows key for the decoder lifetime.
func NewDecoder(key []byte) Decoder {
	return Decoder{src: key}
}

// Reset discards decoder state and borrows key.
func (d *Decoder) Reset(key []byte) {
	d.src = key
	d.off = 0
	d.err = nil
}

// Err returns the first decoding error.
func (d *Decoder) Err() error {
	return d.err
}

// Remaining returns the unconsumed byte count.
func (d *Decoder) Remaining() int {
	return len(d.src) - d.off
}

// Offset returns the next unread byte position.
func (d *Decoder) Offset() int {
	return d.off
}

func (d *Decoder) presence(order Order) (Presence, bool) {
	if d.err != nil {
		return Present, false
	}
	if !order.valid() {
		d.err = ErrInvalidOrder
		return Present, false
	}
	if d.off == len(d.src) {
		d.err = ErrMalformedKey
		return Present, false
	}

	present, null := order.tags()
	tag := d.src[d.off]
	d.off++
	switch tag {
	case present:
		return Present, true
	case null:
		return Null, true
	default:
		d.err = ErrMalformedKey
		return Present, false
	}
}

func (d *Decoder) fixed(order Order, size int, dst []byte) (Presence, bool) {
	presence, ok := d.presence(order)
	if !ok || presence == Null {
		return presence, ok
	}
	if size > len(d.src)-d.off {
		d.err = ErrMalformedKey
		return Present, false
	}

	src := d.src[d.off : d.off+size]
	if order.descending() {
		for i, value := range src {
			dst[i] = ^value
		}
	} else {
		copy(dst, src)
	}
	d.off += size
	return Present, true
}
