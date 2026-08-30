package schottky_test

import (
	"errors"
	"net/netip"
	"testing"

	"gosuda.org/schottky"
)

func TestSemanticScalarsRoundTrip(t *testing.T) {
	uuid := [16]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0xfe, 0xdc, 0xba, 0x98, 0x76, 0x54, 0x32, 0x10}
	mac48 := [6]byte{0, 1, 2, 3, 4, 5}
	mac64 := [8]byte{0, 1, 2, 3, 4, 5, 6, 7}
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Date(-719162, schottky.AscNullsFirst)
		builder.Time(86_399_999_999, schottky.DescNullsLast)
		builder.Timestamp(-1, schottky.AscNullsLast)
		builder.Duration(3_600_000_000, schottky.DescNullsFirst)
		builder.Enum(42, schottky.AscNullsFirst)
		builder.LSN(0x123456789abcdef0, schottky.DescNullsLast)
		builder.UUID(uuid, schottky.AscNullsLast)
		builder.MAC48(mac48, schottky.DescNullsFirst)
		builder.MAC64(mac64, schottky.AscNullsFirst)
	})
	decoder := schottky.NewDecoder(key)
	date, _ := decoder.Date(schottky.AscNullsFirst)
	timeValue, _ := decoder.Time(schottky.DescNullsLast)
	timestamp, _ := decoder.Timestamp(schottky.AscNullsLast)
	duration, _ := decoder.Duration(schottky.DescNullsFirst)
	enum, _ := decoder.Enum(schottky.AscNullsFirst)
	lsn, _ := decoder.LSN(schottky.DescNullsLast)
	decodedUUID, _ := decoder.UUID(schottky.AscNullsLast)
	decodedMAC48, _ := decoder.MAC48(schottky.DescNullsFirst)
	decodedMAC64, _ := decoder.MAC64(schottky.AscNullsFirst)
	if date != -719162 || timeValue != 86_399_999_999 || timestamp != -1 || duration != 3_600_000_000 {
		t.Fatalf("temporal round trip = (%d, %d, %d, %d)", date, timeValue, timestamp, duration)
	}
	if enum != 42 || lsn != 0x123456789abcdef0 {
		t.Fatalf("rank round trip = (%d, %x)", enum, lsn)
	}
	if decodedUUID != uuid || decodedMAC48 != mac48 || decodedMAC64 != mac64 {
		t.Fatalf("identifier round trip = (%x, %x, %x)", decodedUUID, decodedMAC48, decodedMAC64)
	}
	if decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
	}
}

func TestTimeRejectsOutOfRangeValue(t *testing.T) {
	for _, value := range []int64{-1, 86_400_000_000} {
		storage := make([]byte, 0, 16)
		builder := schottky.NewBuilder(storage)
		builder.Time(value, schottky.AscNullsFirst)
		if !errors.Is(builder.Err(), schottky.ErrInvalidValue) {
			t.Fatalf("Time(%d) error = %v, want ErrInvalidValue", value, builder.Err())
		}
	}
}

func TestNetworkValuesRoundTrip(t *testing.T) {
	ipv4 := netip.MustParseAddr("192.0.2.1")
	ipv6 := netip.MustParseAddr("2001:db8::1")
	prefix := netip.MustParsePrefix("192.0.2.129/24")
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(ipv4, schottky.AscNullsFirst)
		builder.IP(ipv6, schottky.DescNullsLast)
		builder.NetworkPrefix(prefix, schottky.AscNullsLast)
	})
	decoder := schottky.NewDecoder(key)
	decodedIPv4, _ := decoder.IP(schottky.AscNullsFirst)
	decodedIPv6, _ := decoder.IP(schottky.DescNullsLast)
	decodedPrefix, _ := decoder.NetworkPrefix(schottky.AscNullsLast)
	if decodedIPv4 != ipv4 || decodedIPv6 != ipv6 {
		t.Fatalf("IP round trip = (%v, %v), want (%v, %v)", decodedIPv4, decodedIPv6, ipv4, ipv6)
	}
	wantPrefix := netip.MustParsePrefix("192.0.2.0/24")
	if decodedPrefix != wantPrefix {
		t.Fatalf("NetworkPrefix() = %v, want %v", decodedPrefix, wantPrefix)
	}
	if decoder.Err() != nil || decoder.Remaining() != 0 {
		t.Fatalf("decoder state: error=%v remaining=%d", decoder.Err(), decoder.Remaining())
	}
}

func TestIPOrdering(t *testing.T) {
	values := []netip.Addr{
		netip.MustParseAddr("0.0.0.0"),
		netip.MustParseAddr("192.0.2.1"),
		netip.MustParseAddr("255.255.255.255"),
		netip.MustParseAddr("::"),
		netip.MustParseAddr("2001:db8::1"),
	}
	keys := make([][]byte, 0, len(values))
	for _, value := range values {
		keys = append(keys, buildKey(t, func(builder *schottky.Builder) {
			builder.IP(value, schottky.AscNullsFirst)
		}))
	}
	assertIncreasing(t, keys)
}

func TestIPNormalizesMappedAddress(t *testing.T) {
	mapped := netip.MustParseAddr("::ffff:192.0.2.1")
	ipv4 := netip.MustParseAddr("192.0.2.1")
	mappedKey := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(mapped, schottky.AscNullsFirst)
	})
	ipv4Key := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(ipv4, schottky.AscNullsFirst)
	})
	if string(mappedKey) != string(ipv4Key) {
		t.Fatalf("mapped key = %x, IPv4 key = %x", mappedKey, ipv4Key)
	}
}

func TestNetworkPrefixNormalizesMappedIPv4(t *testing.T) {
	mapped := netip.MustParsePrefix("::ffff:192.0.2.129/120")
	ipv4 := netip.MustParsePrefix("192.0.2.129/24")
	mappedKey := buildKey(t, func(builder *schottky.Builder) {
		builder.NetworkPrefix(mapped, schottky.AscNullsFirst)
	})
	ipv4Key := buildKey(t, func(builder *schottky.Builder) {
		builder.NetworkPrefix(ipv4, schottky.AscNullsFirst)
	})
	if string(mappedKey) != string(ipv4Key) {
		t.Fatalf("mapped prefix key = %x, IPv4 prefix key = %x", mappedKey, ipv4Key)
	}
}

func TestNetworkRejectsNonPortableValues(t *testing.T) {
	tests := []struct {
		name   string
		encode func(*schottky.Builder)
	}{
		{
			name: "zoned address",
			encode: func(builder *schottky.Builder) {
				builder.IP(netip.MustParseAddr("fe80::1%en0"), schottky.AscNullsFirst)
			},
		},
		{
			name: "zoned mapped IPv4 address",
			encode: func(builder *schottky.Builder) {
				builder.IP(netip.MustParseAddr("::ffff:192.0.2.1%en0"), schottky.AscNullsFirst)
			},
		},
		{
			name: "mapped prefix outside IPv4 bits",
			encode: func(builder *schottky.Builder) {
				builder.NetworkPrefix(netip.MustParsePrefix("::ffff:192.0.2.1/80"), schottky.AscNullsFirst)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var storage [32]byte
			builder := schottky.NewBuilder(storage[:0])
			test.encode(&builder)
			if !errors.Is(builder.Err(), schottky.ErrInvalidValue) {
				t.Fatalf("Err() = %v, want ErrInvalidValue", builder.Err())
			}
			if builder.Len() != 0 {
				t.Fatalf("Len() = %d, want 0", builder.Len())
			}
		})
	}
}

func TestDecoderRejectsNonCanonicalNetworkPrefix(t *testing.T) {
	key := []byte{1, 4, 192, 0, 2, 1, 24}
	decoder := schottky.NewDecoder(key)
	decoder.NetworkPrefix(schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
		t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
	}
}
