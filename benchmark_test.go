package schottky_test

import (
	"testing"

	"gosuda.org/schottky"
)

func BenchmarkCompositeKey(b *testing.B) {
	var storage [256]byte
	builder := schottky.NewBuilder(storage[:0])
	b.ReportAllocs()
	for range b.N {
		builder.Reset(storage[:0])
		builder.Int64(42, schottky.AscNullsLast)
		builder.String("Ada Lovelace", schottky.AscNullsLast)
		builder.Uint64(0x123456789abcdef0, schottky.DescNullsFirst)
		if builder.Err() != nil {
			b.Fatal(builder.Err())
		}
	}
}

func BenchmarkBytesAscending(b *testing.B) {
	value := []byte("prefix/value/without/zero/bytes")
	storage := make([]byte, 0, 2*len(value)+3)
	builder := schottky.NewBuilder(storage)
	b.SetBytes(int64(len(value)))
	b.ReportAllocs()
	for range b.N {
		builder.Reset(storage[:0])
		builder.Bytes(value, schottky.AscNullsFirst)
		if builder.Err() != nil {
			b.Fatal(builder.Err())
		}
	}
}

func BenchmarkBytesDescending(b *testing.B) {
	value := []byte("prefix/value/without/zero/bytes")
	storage := make([]byte, 0, 2*len(value)+3)
	builder := schottky.NewBuilder(storage)
	b.SetBytes(int64(len(value)))
	b.ReportAllocs()
	for range b.N {
		builder.Reset(storage[:0])
		builder.Bytes(value, schottky.DescNullsFirst)
		if builder.Err() != nil {
			b.Fatal(builder.Err())
		}
	}
}

func BenchmarkCompositeDecode(b *testing.B) {
	key := buildKey(b, func(builder *schottky.Builder) {
		builder.Int64(42, schottky.AscNullsLast)
		builder.String("Ada Lovelace", schottky.DescNullsFirst)
		builder.Uint64(0x123456789abcdef0, schottky.AscNullsFirst)
	})
	var storage [64]byte
	b.ReportAllocs()
	for range b.N {
		decoder := schottky.NewDecoder(key)
		decoder.Int64(schottky.AscNullsLast)
		decoder.String(storage[:0], schottky.DescNullsFirst)
		decoder.Uint64(schottky.AscNullsFirst)
		if decoder.Err() != nil {
			b.Fatal(decoder.Err())
		}
	}
}

func BenchmarkCollationKey(b *testing.B) {
	tests := []struct {
		name    string
		profile schottky.CollationProfile
		value   string
	}{
		{
			name:    "root",
			profile: schottky.CollationProfile{AccentSensitive: true, CaseSensitive: true},
			value:   "Straße élan",
		},
		{
			name: "tailored",
			profile: schottky.CollationProfile{
				Tailoring:       schottky.TailoringVietnamese,
				AccentSensitive: true,
				CaseSensitive:   true,
			},
			value: "Việt Nam",
		},
		{
			name:    "binary",
			profile: schottky.CollationProfile{Algorithm: schottky.BinaryCollation},
			value:   "Straße élan",
		},
		{
			name: "simple-case",
			profile: schottky.CollationProfile{
				Algorithm:       schottky.SimpleCaseCollation,
				Padding:         schottky.SpacePadding,
				AccentSensitive: true,
			},
			value: "Straße élan",
		},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			collator, err := schottky.NewCollator(test.profile)
			if err != nil {
				b.Fatal(err)
			}
			var storage [256]byte
			b.SetBytes(int64(len(test.value)))
			b.ReportAllocs()
			for range b.N {
				if _, err := collator.Key(storage[:0], test.value); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
