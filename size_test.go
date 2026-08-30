package schottky_test

import (
	"errors"
	"net/netip"
	"testing"

	"gosuda.org/schottky"
)

func TestEncodedSizesMatchBuilder(t *testing.T) {
	bytesValue := []byte{0, 1, 0}
	bytesSize, err := schottky.EncodedBytesSize(bytesValue)
	if err != nil {
		t.Fatalf("EncodedBytesSize() error = %v", err)
	}
	bytesKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Bytes(bytesValue, schottky.AscNullsFirst)
	})
	if bytesSize != len(bytesKey) {
		t.Fatalf("EncodedBytesSize() = %d, encoded length = %d", bytesSize, len(bytesKey))
	}

	stringSize, err := schottky.EncodedStringSize("A\x00Z")
	if err != nil {
		t.Fatalf("EncodedStringSize() error = %v", err)
	}
	stringKey := buildKey(t, func(builder *schottky.Builder) {
		builder.String("A\x00Z", schottky.DescNullsLast)
	})
	if stringSize != len(stringKey) {
		t.Fatalf("EncodedStringSize() = %d, encoded length = %d", stringSize, len(stringKey))
	}

	decimalSize, err := schottky.EncodedDecimalSize("-00123.4500")
	if err != nil {
		t.Fatalf("EncodedDecimalSize() error = %v", err)
	}
	decimalKey := buildKey(t, func(builder *schottky.Builder) {
		builder.Decimal("-00123.4500", schottky.AscNullsFirst)
	})
	if decimalSize != len(decimalKey) {
		t.Fatalf("EncodedDecimalSize() = %d, encoded length = %d", decimalSize, len(decimalKey))
	}
}

func TestMaxEncodedBinarySize(t *testing.T) {
	size, err := schottky.MaxEncodedBinarySize(3)
	if err != nil || size != 9 {
		t.Fatalf("MaxEncodedBinarySize(3) = (%d, %v), want (9, nil)", size, err)
	}
	if _, err := schottky.MaxEncodedBinarySize(-1); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("MaxEncodedBinarySize(-1) error = %v, want ErrInvalidValue", err)
	}
}

func TestEncodedDecimalSizeRejectsInvalidValue(t *testing.T) {
	if _, err := schottky.EncodedDecimalSize("1e"); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("EncodedDecimalSize() error = %v, want ErrInvalidValue", err)
	}
}

func TestEncodedNetworkSizesMatchBuilder(t *testing.T) {
	address := netip.MustParseAddr("::ffff:192.0.2.1")
	addressSize, err := schottky.EncodedIPSize(address)
	if err != nil {
		t.Fatalf("EncodedIPSize() error = %v", err)
	}
	addressKey := buildKey(t, func(builder *schottky.Builder) {
		builder.IP(address, schottky.DescNullsLast)
	})
	if addressSize != len(addressKey) {
		t.Fatalf("EncodedIPSize() = %d, encoded length = %d", addressSize, len(addressKey))
	}

	ipPrefix := netip.MustParsePrefix("192.0.2.129/24")
	ipPrefixSize, err := schottky.EncodedIPPrefixSize(ipPrefix)
	if err != nil {
		t.Fatalf("EncodedIPPrefixSize() error = %v", err)
	}
	ipPrefixKey := buildKey(t, func(builder *schottky.Builder) {
		builder.IPPrefix(ipPrefix, schottky.AscNullsFirst)
	})
	if ipPrefixSize != len(ipPrefixKey) {
		t.Fatalf("EncodedIPPrefixSize() = %d, encoded length = %d", ipPrefixSize, len(ipPrefixKey))
	}

	network := ipPrefix.Masked()
	networkSize, err := schottky.EncodedNetworkPrefixSize(network)
	if err != nil {
		t.Fatalf("EncodedNetworkPrefixSize() error = %v", err)
	}
	networkKey := buildKey(t, func(builder *schottky.Builder) {
		builder.NetworkPrefix(network, schottky.AscNullsFirst)
	})
	if networkSize != len(networkKey) {
		t.Fatalf("EncodedNetworkPrefixSize() = %d, encoded length = %d", networkSize, len(networkKey))
	}
}
