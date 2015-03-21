package swim

var deBruijn = []int{0, 1, 28, 2, 29, 14, 24, 3, 30, 22, 20, 15, 25, 17, 4,
	8, 31, 27, 13, 23, 21, 19, 16, 7, 26, 12, 18, 6, 11, 5, 10, 9}

// https://graphics.stanford.edu/~seander/bithacks.html#IntegerLogDeBruijn
func log2ceil(x int) int {
	v := uint32(x)

	// round to next power of 2
	if v&(v-1) != 0 {
		v |= v >> 1
		v |= v >> 2
		v |= v >> 4
		v |= v >> 8
		v |= v >> 16
		v += 1
	}

	return deBruijn[(v*0x077CB531)>>27]
}
