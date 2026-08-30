package schottky

// PrefixUpperBound appends the shortest exclusive upper bound for prefix to dst.
// ok is false when prefix has no finite upper bound; dst is then unchanged.
func PrefixUpperBound(dst, prefix []byte) (bound []byte, ok bool, err error) {
	last := len(prefix) - 1
	for last >= 0 && prefix[last] == 0xff {
		last--
	}
	if last < 0 {
		return dst, false, nil
	}
	needed := last + 1
	if needed > cap(dst)-len(dst) {
		return dst, false, ErrShortBuffer
	}

	start := len(dst)
	dst = dst[:start+needed]
	copy(dst[start:], prefix[:needed])
	dst[len(dst)-1]++
	return dst, true, nil
}
