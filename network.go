package schottky

import (
	"net/netip"

	"gosuda.org/schottky/internal/byteops"
)

// IP appends an unzoned address while preserving its address family.
func (b *Builder) IP(value netip.Addr, order Order) {
	if !value.IsValid() || value.Zone() != "" {
		b.fail(ErrInvalidValue)
		return
	}
	b.networkPrefix(netip.PrefixFrom(value, value.BitLen()), order)
}

// IPPrefix appends an unzoned address and prefix length while preserving host bits.
func (b *Builder) IPPrefix(value netip.Prefix, order Order) {
	if !value.IsValid() || value.Addr().Zone() != "" {
		b.fail(ErrInvalidValue)
		return
	}
	b.networkPrefix(value, order)
}

// NetworkPrefix appends an unzoned canonical network prefix.
func (b *Builder) NetworkPrefix(value netip.Prefix, order Order) {
	if !value.IsValid() || value.Addr().Zone() != "" || value.Masked() != value {
		b.fail(ErrInvalidValue)
		return
	}
	b.networkPrefix(value, order)
}

func (b *Builder) networkPrefix(value netip.Prefix, order Order) {
	addressValue := value.Addr()
	bits := value.Bits()
	if addressValue.Is4() {
		address := addressValue.As4()
		b.networkBytes(4, address[:], bits, order)
		return
	}
	address := addressValue.As16()
	b.networkBytes(6, address[:], bits, order)
}

func (b *Builder) networkBytes(family byte, address []byte, bits int, order Order) {
	prefixSize := (bits + 7) / 8
	escapedSize := 2
	for i := range prefixSize {
		if maskedAddressByte(address, i, bits) == 0 {
			escapedSize++
		}
		escapedSize++
	}

	payload, ok := b.begin(order, 1+escapedSize+1+len(address))
	if !ok {
		return
	}
	payload[0] = family
	out := 1
	for i := range prefixSize {
		valueByte := maskedAddressByte(address, i, bits)
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
	out += 2
	payload[out] = byte(bits)
	out++
	copy(payload[out:], address)
	if order.descending() {
		byteops.Invert(payload)
	}
}

func maskedAddressByte(address []byte, index, bits int) byte {
	value := address[index]
	if remaining := bits & 7; index == bits/8 && remaining != 0 {
		value &= byte(0xff << (8 - remaining))
	}
	return value
}

// EncodedIPSize returns the exact field size for value, including presence.
func EncodedIPSize(value netip.Addr) (int, error) {
	if !value.IsValid() || value.Zone() != "" {
		return 0, ErrInvalidValue
	}
	return encodedNetworkPrefixSize(netip.PrefixFrom(value, value.BitLen()))
}

// EncodedIPPrefixSize returns the exact field size for value, including presence.
func EncodedIPPrefixSize(value netip.Prefix) (int, error) {
	if !value.IsValid() || value.Addr().Zone() != "" {
		return 0, ErrInvalidValue
	}
	return encodedNetworkPrefixSize(value)
}

// EncodedNetworkPrefixSize returns the exact field size for a canonical value.
func EncodedNetworkPrefixSize(value netip.Prefix) (int, error) {
	if !value.IsValid() || value.Addr().Zone() != "" || value.Masked() != value {
		return 0, ErrInvalidValue
	}
	return encodedNetworkPrefixSize(value)
}

func encodedNetworkPrefixSize(value netip.Prefix) (int, error) {
	addressValue := value.Addr()
	bits := value.Bits()
	addressSize := 16
	var address [16]byte
	if addressValue.Is4() {
		addressSize = 4
		address4 := addressValue.As4()
		copy(address[:4], address4[:])
	} else {
		address = addressValue.As16()
	}
	prefixSize := (bits + 7) / 8
	escapedSize := 2 + prefixSize
	for i := range prefixSize {
		if maskedAddressByte(address[:addressSize], i, bits) == 0 {
			escapedSize++
		}
	}
	return 1 + 1 + escapedSize + 1 + addressSize, nil
}

func (d *Decoder) IP(order Order) (netip.Addr, Presence) {
	prefix, presence := d.IPPrefix(order)
	if d.err != nil || presence == Null {
		return netip.Addr{}, presence
	}
	if prefix.Bits() != prefix.Addr().BitLen() {
		d.err = ErrMalformedKey
		return netip.Addr{}, Present
	}
	return prefix.Addr(), Present
}

func (d *Decoder) IPPrefix(order Order) (netip.Prefix, Presence) {
	presence, ok := d.presence(order)
	if !ok || presence == Null {
		return netip.Prefix{}, presence
	}
	if d.off == len(d.src) {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}

	descending := order.descending()
	family := d.src[d.off]
	if descending {
		family = ^family
	}
	addressSize := 0
	maxBits := 0
	switch family {
	case 4:
		addressSize, maxBits = 4, 32
	case 6:
		addressSize, maxBits = 16, 128
	default:
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}

	prefixStart := d.off + 1
	prefixEnd, prefixSize, ok := scanEscaped(d.src[prefixStart:], descending)
	if !ok {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}
	bitsOffset := prefixStart + prefixEnd
	if 1+addressSize > len(d.src)-bitsOffset {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}
	bitsByte := d.src[bitsOffset]
	if descending {
		bitsByte = ^bitsByte
	}
	bits := int(bitsByte)
	if bits > maxBits || prefixSize != (bits+7)/8 {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}

	var addressBytes [16]byte
	addressOffset := bitsOffset + 1
	for i := range addressSize {
		valueByte := d.src[addressOffset+i]
		if descending {
			valueByte = ^valueByte
		}
		addressBytes[i] = valueByte
	}
	if !d.validNetworkPrefix(
		d.src[prefixStart:bitsOffset],
		addressBytes[:addressSize],
		bits,
		descending,
	) {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}

	var address netip.Addr
	if family == 4 {
		address = netip.AddrFrom4([4]byte(addressBytes[:4]))
	} else {
		address = netip.AddrFrom16(addressBytes)
	}
	d.off = addressOffset + addressSize
	return netip.PrefixFrom(address, bits), Present
}

func (d *Decoder) NetworkPrefix(order Order) (netip.Prefix, Presence) {
	prefix, presence := d.IPPrefix(order)
	if d.err != nil || presence == Null {
		return prefix, presence
	}
	if prefix.Masked() != prefix {
		d.err = ErrMalformedKey
		return netip.Prefix{}, Present
	}
	return prefix, Present
}

func (d *Decoder) validNetworkPrefix(encoded, address []byte, bits int, descending bool) bool {
	sentinel := byte(0)
	if descending {
		sentinel = 0xff
	}
	out := 0
	for i := 0; i < len(encoded)-2; {
		valueByte := encoded[i]
		if valueByte == sentinel {
			valueByte = 0
			i += 2
		} else {
			if descending {
				valueByte = ^valueByte
			}
			i++
		}
		if valueByte != maskedAddressByte(address, out, bits) {
			return false
		}
		out++
	}
	return out == (bits+7)/8
}
