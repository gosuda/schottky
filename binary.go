package schottky

import "gosuda.org/schottky/internal/byteops"

// Bytes appends an unsigned bytewise-ordered binary field.
func (b *Builder) Bytes(value []byte, order Order) {
	payloadSize, ok := escapedBytesSize(value)
	if !ok {
		b.fail(ErrShortBuffer)
		return
	}
	payload, ok := b.begin(order, payloadSize)
	if !ok {
		return
	}

	out := 0
	for _, valueByte := range value {
		if valueByte == 0 {
			payload[out] = 0
			payload[out+1] = 0xff
			out += 2
			continue
		}
		payload[out] = valueByte
		out++
	}
	payload[out] = 0
	payload[out+1] = 0
	if order.descending() {
		byteops.Invert(payload)
	}
}

// String appends a binary-ordered string field without converting it to []byte.
func (b *Builder) String(value string, order Order) {
	payloadSize, ok := escapedStringSize(value)
	if !ok {
		b.fail(ErrShortBuffer)
		return
	}
	payload, ok := b.begin(order, payloadSize)
	if !ok {
		return
	}

	out := 0
	for i := 0; i < len(value); i++ {
		valueByte := value[i]
		if valueByte == 0 {
			payload[out] = 0
			payload[out+1] = 0xff
			out += 2
			continue
		}
		payload[out] = valueByte
		out++
	}
	payload[out] = 0
	payload[out+1] = 0
	if order.descending() {
		byteops.Invert(payload)
	}
}

// CollationKey appends an externally generated collation key.
func (b *Builder) CollationKey(value []byte, order Order) {
	b.Bytes(value, order)
}

// Tuple appends an already encoded nested key.
func (b *Builder) Tuple(value []byte, order Order) {
	b.Bytes(value, order)
}

// Bytes decodes a binary field into caller-owned capacity after len(dst).
func (d *Decoder) Bytes(dst []byte, order Order) ([]byte, Presence) {
	presence, ok := d.presence(order)
	if !ok || presence == Null {
		return dst, presence
	}

	end, decodedSize, ok := scanEscaped(d.src[d.off:], order.descending())
	if !ok {
		d.err = ErrMalformedKey
		return dst, Present
	}
	if decodedSize > cap(dst)-len(dst) {
		d.err = ErrShortBuffer
		return dst, Present
	}

	start := len(dst)
	dst = dst[:start+decodedSize]
	encoded := d.src[d.off : d.off+end-2]
	out := start
	sentinel := byte(0)
	if order.descending() {
		sentinel = 0xff
	}
	for i := 0; i < len(encoded); {
		valueByte := encoded[i]
		if valueByte == sentinel {
			dst[out] = 0
			out++
			i += 2
			continue
		}
		if order.descending() {
			valueByte = ^valueByte
		}
		dst[out] = valueByte
		out++
		i++
	}
	d.off += end
	return dst, Present
}

// String decodes a binary string into caller-owned byte capacity.
func (d *Decoder) String(dst []byte, order Order) ([]byte, Presence) {
	return d.Bytes(dst, order)
}

// CollationKey decodes an opaque collation key.
func (d *Decoder) CollationKey(dst []byte, order Order) ([]byte, Presence) {
	return d.Bytes(dst, order)
}

// Tuple decodes an embedded key.
func (d *Decoder) Tuple(dst []byte, order Order) ([]byte, Presence) {
	return d.Bytes(dst, order)
}

func escapedBytesSize(value []byte) (int, bool) {
	size := len(value) + 2
	if size < len(value) {
		return 0, false
	}
	for _, valueByte := range value {
		if valueByte != 0 {
			continue
		}
		if size == int(^uint(0)>>1) {
			return 0, false
		}
		size++
	}
	return size, true
}

func escapedStringSize(value string) (int, bool) {
	size := len(value) + 2
	if size < len(value) {
		return 0, false
	}
	for i := 0; i < len(value); i++ {
		if value[i] != 0 {
			continue
		}
		if size == int(^uint(0)>>1) {
			return 0, false
		}
		size++
	}
	return size, true
}

func scanEscaped(src []byte, descending bool) (end, decodedSize int, ok bool) {
	sentinel, escaped, terminal := byte(0), byte(0xff), byte(0)
	if descending {
		sentinel, escaped, terminal = 0xff, 0, 0xff
	}
	for i := 0; i < len(src); {
		if src[i] != sentinel {
			decodedSize++
			i++
			continue
		}
		if i+1 == len(src) {
			return 0, 0, false
		}
		switch src[i+1] {
		case terminal:
			return i + 2, decodedSize, true
		case escaped:
			decodedSize++
			i += 2
		default:
			return 0, 0, false
		}
	}
	return 0, 0, false
}
