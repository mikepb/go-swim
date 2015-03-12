package swim

import (
	"testing"
)

func testListSize(t *testing.T, l SelectionList, n int) {
	if s := l.Size(); s != n {
		t.Fatalf("expected size %v got %v", n, s)
	} else if s := len(l.List()); s != n {
		t.Fatalf("expected size %v got %v", n, s)
	}
}
