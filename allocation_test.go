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

func TestVectorProjectionDoesNotAllocate(t *testing.T) {
	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionGaussian,
		schottky.VectorL2,
		4,
		nil,
		[]float32{1, 0, 0, 0, 0, 1, 0, 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	vector := []float32{1, 2, 3, 4}
	failed := false
	allocations := testing.AllocsPerRun(1000, func() {
		var storage [2]float32
		_, projectErr := profile.Project(storage[:0], vector)
		failed = failed || projectErr != nil
	})
	if failed {
		t.Fatal("vector projection returned an error")
	}
	if allocations != 0 {
		t.Fatalf("vector projection allocations = %v, want 0", allocations)
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

func TestCollationEncodingDoesNotAllocate(t *testing.T) {
	tests := []struct {
		name    string
		profile schottky.CollationProfile
		text    string
	}{
		{
			name:    "root",
			profile: schottky.CollationProfile{AccentSensitive: true, CaseSensitive: true},
			text:    "Straße élan",
		},
		{
			name: "tailored",
			profile: schottky.CollationProfile{
				Tailoring:       schottky.TailoringVietnamese,
				AccentSensitive: true,
				CaseSensitive:   true,
			},
			text: "Việt Nam",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collator, err := schottky.NewCollator(test.profile)
			if err != nil {
				t.Fatalf("NewCollator() error = %v", err)
			}
			failed := false
			allocations := testing.AllocsPerRun(1000, func() {
				var storage [256]byte
				_, keyErr := collator.Key(storage[:0], test.text)
				failed = failed || keyErr != nil
			})
			if failed {
				t.Fatal("collation encoding returned an error")
			}
			if allocations != 0 {
				t.Fatalf("collation encoding allocations = %v, want 0", allocations)
			}
		})
	}
}
