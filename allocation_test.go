package schottky_test

import (
	"net/netip"
	"testing"

	"gosuda.org/schottky"
)

func TestCoreEncodingDoesNotAllocate(t *testing.T) {
	address := netip.MustParseAddr("2001:db8::1")
	failed := false
	allocations := testing.AllocsPerRun(1000, func() {
		var storage [512]byte
		builder := schottky.NewBuilder(storage[:0])
		builder.Int64(-42, schottky.AscNullsLast)
		builder.Uint64(42, schottky.DescNullsFirst)
		builder.Float64(3.5, schottky.AscNullsFirst)
		builder.String("A\x00Z", schottky.DescNullsLast)
		builder.Decimal("-12345.6789e2", schottky.AscNullsLast)
		builder.IP(address, schottky.DescNullsFirst)
		failed = failed || builder.Err() != nil
	})
	if failed {
		t.Fatal("encoding returned an error")
	}
	if allocations != 0 {
		t.Fatalf("encoding allocations = %v, want 0", allocations)
	}
}

func TestCoreDecodingDoesNotAllocate(t *testing.T) {
	key := buildKey(t, func(builder *schottky.Builder) {
		builder.Int64(-42, schottky.AscNullsLast)
		builder.String("A\x00Z", schottky.DescNullsLast)
		builder.Decimal("-12345.6789e2", schottky.AscNullsLast)
	})
	failed := false
	allocations := testing.AllocsPerRun(1000, func() {
		decoder := schottky.NewDecoder(key)
		var storage [64]byte
		decoder.Int64(schottky.AscNullsLast)
		decoded := storage[:0]
		decoded, _ = decoder.String(decoded, schottky.DescNullsLast)
		decoder.Decimal(decoded, schottky.AscNullsLast)
		failed = failed || decoder.Err() != nil
	})
	if failed {
		t.Fatal("decoding returned an error")
	}
	if allocations != 0 {
		t.Fatalf("decoding allocations = %v, want 0", allocations)
	}
}
