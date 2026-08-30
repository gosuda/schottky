//go:build !goexperiment.simd || (!amd64 && !arm64 && !wasm)

package byteops

func Invert(data []byte) {
	for i := range data {
		data[i] = ^data[i]
	}
}
