package swim

import (
	"testing"
)

func TestLog2ceil(t *testing.T) {
	fixtures := map[int]int{
		1:    0,
		2:    1,
		3:    2,
		4:    2,
		5:    3,
		7:    3,
		8:    3,
		9:    4,
		15:   4,
		16:   4,
		4095: 12,
		4096: 12,
		4097: 13,
	}
	for subject, expect := range fixtures {
		t.Logf("log(%v) = %v", subject, expect)
		if actual := log2ceil(subject); actual != expect {
			t.Fatalf("expected log2ceil(%v) = %v got %v", subject, expect, actual)
		}
	}
}
