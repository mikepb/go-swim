package swim

import (
	"testing"
)

func testListLen(t *testing.T, l SelectionList, n int) {
	if s := l.Len(); s != n {
		t.Fatalf("expected size %v got %v", n, s)
	} else if s := len(l.List()); s != n {
		t.Fatalf("expected size %v got %v", n, s)
	}
}
