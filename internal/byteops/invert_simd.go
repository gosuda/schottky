//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package byteops

import "simd"

func Invert(data []byte) {
	width := simd.BroadcastUint8s(0).Len()
	for len(data) >= width {
		simd.LoadUint8s(data).Not().Store(data)
		data = data[width:]
	}
	for i := range data {
		data[i] = ^data[i]
	}
}
