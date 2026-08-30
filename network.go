package schottky

import (
	"net/netip"

	"gosuda.org/schottky/internal/byteops"
)

// IP appends an unzoned IP address, normalizing mapped IPv4 addresses.
func (b *Builder) IP(value netip.Addr, order Order) {
	if !value.IsValid() || value.Zone() != "" {
		b.fail(ErrInvalidValue)
		return
	}
	value = value.Unmap()
	if value.Is4() {
		address := value.As4()
		b.networkBytes(4, address[:], 0, false, order)
		return
	}
	address := value.As16()
	b.networkBytes(6, address[:], 0, false, order)
}

// NetworkPrefix appends a masked, unzoned IP network and prefix length.
func (b *Builder) NetworkPrefix(value netip.Prefix, order Order) {
	if !value.IsValid() || value.Addr().Zone() != "" {
		b.fail(ErrInvalidValue)
		return
	}

	addressValue := value.Addr()
	bits := value.Bits()
	if addressValue.Is4In6() {
		if bits < 96 {
			b.fail(ErrInvalidValue)
			return
		}
		addressValue = addressValue.Unmap()
		bits -= 96
	}
	value = netip.PrefixFrom(addressValue, bits).Masked()
	addressValue = value.Addr()
	if addressValue.Is4() {
		address := addressValue.As4()
		b.networkBytes(4, address[:], byte(bits), true, order)
		return
	}
	address := addressValue.As16()
	b.networkBytes(6, address[:], byte(bits), true, order)
}

func (b *Builder) networkBytes(family byte, address []byte, bits byte, includeBits bool, order Order) {
	payloadSize := 1 + len(address)
	if includeBits {
		payloadSize++
	}
	payload, ok := b.begin(order, payloadSize)
	if !ok {
		return
	}
	payload[0] = family
	copy(payload[1:], address)
	if includeBits {
		payload[len(payload)-1] = bits
	}
	if order.descending() {
		byteops.Invert(payload)
	}
}

func (d *Decoder) IP(order Order) (netip.Addr, Presence) {
	address, _, presence, ok := d.network(order, false)
	if !ok || presence == Null {
		return netip.Addr{}, presence
	}
	return address, Present
}

func (d *Decoder) NetworkPrefix(order Order) (netip.Prefix, Presence) {
	address, bits, presence, ok := d.network(order, true)
	if !ok || presence == Null {
		return netip.Prefix{}, presence
	}
	prefix := netip.PrefixFrom(address, bits)
	if !prefix.IsValid() || prefix.Masked().Addr() != address {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}
	return prefix, Present
}

func (d *Decoder) network(order Order, includeBits bool) (netip.Addr, int, Presence, bool) {
	presence, ok := d.presence(order)
	if !ok || presence == Null {
		return netip.Addr{}, 0, presence, ok
	}
	if d.off == len(d.src) {
		d.err = ErrMalformedKey
		return netip.Addr{}, 0, Present, false
	}
	family := d.src[d.off]
	if order.descending() {
		family = ^family
	}
	addressSize := 0
	switch family {
	case 4:
		addressSize = 4
	case 6:
		addressSize = 16
	default:
		d.err = ErrMalformedKey
		return netip.Addr{}, 0, Present, false
	}
	payloadSize := 1 + addressSize
	if includeBits {
		payloadSize++
	}
	if payloadSize > len(d.src)-d.off {
		d.err = ErrMalformedKey
		return netip.Addr{}, 0, Present, false
	}

	var raw [16]byte
	for i := range addressSize {
		valueByte := d.src[d.off+1+i]
		if order.descending() {
			valueByte = ^valueByte
		}
		raw[i] = valueByte
	}
	var address netip.Addr
	if family == 4 {
		address = netip.AddrFrom4([4]byte(raw[:4]))
	} else {
		address = netip.AddrFrom16(raw)
	}
	bits := 0
	if includeBits {
		bitsByte := d.src[d.off+payloadSize-1]
		if order.descending() {
			bitsByte = ^bitsByte
		}
		bits = int(bitsByte)
		maxBits := 128
		if family == 4 {
			maxBits = 32
		}
		if bits > maxBits {
			d.err = ErrMalformedKey
			return netip.Addr{}, 0, Present, false
		}
	}
	d.off += payloadSize
	return address, bits, Present, true
}
