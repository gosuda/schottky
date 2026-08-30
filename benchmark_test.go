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
