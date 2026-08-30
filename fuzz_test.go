package schottky_test

import (
	"bytes"
	"testing"

	"gosuda.org/schottky"
)

func FuzzBytesRoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 0xff, 0})
	f.Fuzz(func(t *testing.T, value []byte) {
		orders := [...]schottky.Order{
			schottky.AscNullsFirst,
			schottky.AscNullsLast,
			schottky.DescNullsFirst,
			schottky.DescNullsLast,
		}
		for _, order := range orders {
			storage := make([]byte, 2*len(value)+3)
			builder := schottky.NewBuilder(storage[:0])
			builder.Bytes(value, order)
			key, err := builder.Key()
			if err != nil {
				t.Fatalf("Bytes(%x, %v): %v", value, order, err)
			}

			decoder := schottky.NewDecoder(key)
			decodedStorage := make([]byte, len(value))
			decoded, presence := decoder.Bytes(decodedStorage[:0], order)
			if decoder.Err() != nil {
				t.Fatalf("decode Bytes(%x, %v): %v", value, order, decoder.Err())
			}
			if presence != schottky.Present || !bytes.Equal(decoded, value) || decoder.Remaining() != 0 {
				t.Fatalf("round trip = (%x, %v, remaining %d), want (%x, Present, 0)", decoded, presence, decoder.Remaining(), value)
			}
		}
	})
}

func FuzzCompositeOrdering(f *testing.F) {
	f.Add(int64(1), "Ada", int64(2), "Bob")
	f.Add(int64(-1), "\x00", int64(-1), "\x00A")
	f.Fuzz(func(t *testing.T, leftInt int64, leftString string, rightInt int64, rightString string) {
		encode := func(integer int64, text string) []byte {
			storage := make([]byte, 0, 12+2*len(text))
			builder := schottky.NewBuilder(storage)
			builder.Int64(integer, schottky.AscNullsFirst)
			builder.String(text, schottky.AscNullsFirst)
			key, err := builder.Key()
			if err != nil {
				t.Fatalf("build composite key: %v", err)
			}
			return key
		}
		left := encode(leftInt, leftString)
		right := encode(rightInt, rightString)
		want := 0
		switch {
		case leftInt < rightInt:
			want = -1
		case leftInt > rightInt:
			want = 1
		default:
			want = bytes.Compare([]byte(leftString), []byte(rightString))
		}
		got := bytes.Compare(left, right)
		if (got < 0) != (want < 0) || (got > 0) != (want > 0) {
			t.Fatalf("key comparison = %d, value comparison = %d; left=(%d,%x) right=(%d,%x)", got, want, leftInt, leftString, rightInt, rightString)
		}
	})
}

func FuzzDecimalCanonicalRoundTrip(f *testing.F) {
	f.Add("0")
	f.Add("-00123.4500e+2")
	f.Add("Infinity")
	f.Add("NaN")
	f.Fuzz(func(t *testing.T, value string) {
		size, err := schottky.EncodedDecimalSize(value)
		if err != nil {
			return
		}
		storage := make([]byte, size)
		builder := schottky.NewBuilder(storage[:0])
		builder.Decimal(value, schottky.DescNullsLast)
		key, err := builder.Key()
		if err != nil {
			t.Fatalf("Decimal(%q): %v", value, err)
		}

		decodedStorage := make([]byte, 0, len(value)+16)
		decoder := schottky.NewDecoder(key)
		decoded, presence := decoder.Decimal(decodedStorage, schottky.DescNullsLast)
		if decoder.Err() != nil || presence != schottky.Present || decoder.Remaining() != 0 {
			t.Fatalf("decode Decimal(%q) = (%q, %v), error=%v remaining=%d", value, decoded, presence, decoder.Err(), decoder.Remaining())
		}

		reencodedStorage := make([]byte, size)
		reencodedBuilder := schottky.NewBuilder(reencodedStorage[:0])
		reencodedBuilder.Decimal(string(decoded), schottky.DescNullsLast)
		reencoded, err := reencodedBuilder.Key()
		if err != nil {
			t.Fatalf("re-encode Decimal(%q): %v", decoded, err)
		}
		if !bytes.Equal(reencoded, key) {
			t.Fatalf("re-encoded key = %x, want %x for %q", reencoded, key, value)
		}
	})
}
