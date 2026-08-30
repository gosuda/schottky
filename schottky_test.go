package schottky_test

import (
	"bytes"
	"testing"

	"gosuda.org/schottky"
)

func buildKey(t testing.TB, encode func(*schottky.Builder)) []byte {
	t.Helper()
	storage := make([]byte, 0, 4096)
	builder := schottky.NewBuilder(storage)
	encode(&builder)
	key, err := builder.Key()
	if err != nil {
		t.Fatalf("build key: %v", err)
	}
	return bytes.Clone(key)
}

func assertIncreasing(t testing.TB, keys [][]byte) {
	t.Helper()
	for i := 1; i < len(keys); i++ {
		if bytes.Compare(keys[i-1], keys[i]) >= 0 {
			t.Fatalf("keys[%d] = %x, keys[%d] = %x; want strict increase", i-1, keys[i-1], i, keys[i])
		}
	}
}

func assertDecreasingValues(t testing.TB, keys [][]byte) {
	t.Helper()
	for i := 1; i < len(keys); i++ {
		if bytes.Compare(keys[i-1], keys[i]) <= 0 {
			t.Fatalf("keys[%d] = %x, keys[%d] = %x; want strict decrease", i-1, keys[i-1], i, keys[i])
		}
	}
}
