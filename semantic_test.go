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
		builder.Time(86_400_000_000, schottky.DescNullsLast)
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
	if date != -719162 || timeValue != 86_400_000_000 || timestamp != -1 || duration != 3_600_000_000 {
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
	for _, value := range []int64{-1, 86_400_000_001} {
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
	ipPrefix := netip.MustParsePrefix("192.0.2.129/24")
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(ipv4, schottky.AscNullsFirst)
		builder.IP(ipv6, schottky.DescNullsLast)
		builder.IPPrefix(ipPrefix, schottky.AscNullsLast)
	})
	decoder := schottky.NewDecoder(key)
	decodedIPv4, _ := decoder.IP(schottky.AscNullsFirst)
	decodedIPv6, _ := decoder.IP(schottky.DescNullsLast)
	decodedPrefix, _ := decoder.IPPrefix(schottky.AscNullsLast)
	if decodedIPv4 != ipv4 || decodedIPv6 != ipv6 {
		t.Fatalf("IP round trip = (%v, %v), want (%v, %v)", decodedIPv4, decodedIPv6, ipv4, ipv6)
	}
	if decodedPrefix != ipPrefix {
		t.Fatalf("IPPrefix() = %v, want %v", decodedPrefix, ipPrefix)
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

func TestIPPreservesMappedAddressFamily(t *testing.T) {
	mapped := netip.MustParseAddr("::ffff:192.0.2.1")
	ipv4 := netip.MustParseAddr("192.0.2.1")
	mappedKey := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(mapped, schottky.AscNullsFirst)
	})
	ipv4Key := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(ipv4, schottky.AscNullsFirst)
	})
	if string(ipv4Key) >= string(mappedKey) {
		t.Fatalf("IPv4 key = %x, mapped IPv6 key = %x", ipv4Key, mappedKey)
	}
	decoder := schottky.NewDecoder(mappedKey)
	decoded, _ := decoder.IP(schottky.AscNullsFirst)
	if decoded != mapped || decoder.Err() != nil {
		t.Fatalf("IP() = (%v, %v), want (%v, nil)", decoded, decoder.Err(), mapped)
	}
}

func TestIPPrefixPreservesMappedAddressFamilyAndHostBits(t *testing.T) {
	mapped := netip.MustParsePrefix("::ffff:192.0.2.129/120")
	ipv4 := netip.MustParsePrefix("192.0.2.129/24")
	mappedKey := buildKey(t, func(builder *schottky.Builder) {
		builder.IPPrefix(mapped, schottky.AscNullsFirst)
	})
	ipv4Key := buildKey(t, func(builder *schottky.Builder) {
		builder.IPPrefix(ipv4, schottky.AscNullsFirst)
	})
	if string(ipv4Key) >= string(mappedKey) {
		t.Fatalf("IPv4 prefix key = %x, mapped IPv6 prefix key = %x", ipv4Key, mappedKey)
	}
	decoder := schottky.NewDecoder(mappedKey)
	decoded, _ := decoder.IPPrefix(schottky.AscNullsFirst)
	if decoded != mapped || decoder.Err() != nil {
		t.Fatalf("IPPrefix() = (%v, %v), want (%v, nil)", decoded, decoder.Err(), mapped)
	}
}

func TestNetworkRejectsZonedValues(t *testing.T) {
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

func TestIPPrefixOrdering(t *testing.T) {
	values := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.1/8"),
		netip.MustParsePrefix("10.0.0.0/16"),
		netip.MustParsePrefix("10.0.0.1/16"),
		netip.MustParsePrefix("10.0.0.0/24"),
		netip.MustParsePrefix("::/0"),
		netip.MustParsePrefix("::ffff:10.0.0.0/120"),
	}
	keys := make([][]byte, 0, len(values))
	for _, value := range values {
		keys = append(keys, buildKey(t, func(builder *schottky.Builder) {
			builder.IPPrefix(value, schottky.AscNullsFirst)
		}))
	}
	assertIncreasing(t, keys)
}

func TestDecoderRejectsMismatchedIPPrefix(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.IPPrefix(netip.MustParsePrefix("192.0.2.1/24"), schottky.AscNullsFirst)
	})
	key[2] ^= 1
	decoder := schottky.NewDecoder(key)
	decoder.IPPrefix(schottky.AscNullsFirst)
	if !errors.Is(decoder.Err(), schottky.ErrMalformedKey) {
		t.Fatalf("Err() = %v, want ErrMalformedKey", decoder.Err())
	}
}

func TestNetworkPrefixRejectsHostBits(t *testing.T) {
	var storage [64]byte
	builder := schottky.NewBuilder(storage[:0])
	builder.NetworkPrefix(netip.MustParsePrefix("192.0.2.1/24"), schottky.AscNullsFirst)
	if !errors.Is(builder.Err(), schottky.ErrInvalidValue) {
		t.Fatalf("Err() = %v, want ErrInvalidValue", builder.Err())
	}
}

func TestZonedTimeOrderingAndRoundTrip(t *testing.T) {
	first := schottky.ZonedTime{Microseconds: 10 * 3_600_000_000, UTCOffsetSeconds: 2 * 60 * 60}
	second := schottky.ZonedTime{Microseconds: 8 * 3_600_000_000}
	firstKey := buildKey(t, func(builder *schottky.Builder) {
		builder.ZonedTime(first, schottky.AscNullsFirst)
	})
	secondKey := buildKey(t, func(builder *schottky.Builder) {
		builder.ZonedTime(second, schottky.AscNullsFirst)
	})
	if string(firstKey) >= string(secondKey) {
		t.Fatalf("equal-instant keys = (%x, %x), want first < second", firstKey, secondKey)
	}
	decoder := schottky.NewDecoder(firstKey)
	decoded, _ := decoder.ZonedTime(schottky.AscNullsFirst)
	if decoded != first || decoder.Err() != nil {
		t.Fatalf("ZonedTime() = (%v, %v), want (%v, nil)", decoded, decoder.Err(), first)
	}
}

func TestIntervalOrderValue(t *testing.T) {
	oneMonth := schottky.IntervalOrderValue(1, 0, 0)
	thirtyDays := schottky.IntervalOrderValue(0, 30, 0)
	if oneMonth != thirtyDays {
		t.Fatalf("one month = %v, thirty days = %v", oneMonth, thirtyDays)
	}
	zeroKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Int128(schottky.IntervalOrderValue(0, 0, 0), schottky.AscNullsFirst)
	})
	oneDayKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Int128(schottky.IntervalOrderValue(-1, 31, 0), schottky.AscNullsFirst)
	})
	if string(zeroKey) >= string(oneDayKey) {
		t.Fatalf("interval keys = (%x, %x), want zero < one day", zeroKey, oneDayKey)
	}
}

func TestTemporalScalarsRejectValuesOutsideFiniteDomains(t *testing.T) {
	tests := []struct {
		name   string
		encode func(*schottky.Builder)
	}{
		{name: "date below finite range", encode: func(builder *schottky.Builder) {
			builder.Date(-2_451_546, schottky.AscNullsFirst)
		}},
		{name: "date at finite end", encode: func(builder *schottky.Builder) {
			builder.Date(2_145_031_949, schottky.AscNullsFirst)
		}},
		{name: "timestamp below finite range", encode: func(builder *schottky.Builder) {
			builder.Timestamp(-211_813_488_000_000_001, schottky.AscNullsFirst)
		}},
		{name: "timestamp at finite end", encode: func(builder *schottky.Builder) {
			builder.Timestamp(9_223_371_331_200_000_000, schottky.AscNullsFirst)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var storage [32]byte
			builder := schottky.NewBuilder(storage[:0])
			test.encode(&builder)
			if !errors.Is(builder.Err(), schottky.ErrInvalidValue) {
				t.Fatalf("Err() = %v, want ErrInvalidValue", builder.Err())
			}
		})
	}
}
