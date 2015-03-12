package swim

import (
	"testing"
)

func TestShuffleList(t *testing.T) {
	var sl ShuffleList

	if len(sl.List()) != 0 {
		t.Fatalf("expected empty list")
	} else if sl.Next() != nil {
		t.Fatalf("expected nil next")
	}

	n1 := &InternalNode{Node: Node{Id: 1}}
	n2 := &InternalNode{Node: Node{Id: 2}}

	sl.Add(n1, n2)
	if l := sl.List(); len(l) != 2 {
		t.Fatalf("expected 2 nodes, got %v", len(l))
	} else if l[0] != n1 {
		t.Fatalf("expected node 1, got %v", l[0])
	} else if l[1] != n2 {
		t.Fatalf("expected node 2, got %v", l[1])
	}

	if n := sl.Next(); n != n1 {
		t.Fatalf("expected node 1, got %v", n)
	} else if n := sl.Next(); n != n2 {
		t.Fatalf("expected node 2, got %v", n)
	}

	for i := 3; i <= 999; i += 1 {
		sl.Add(&InternalNode{Node: Node{Id: uint64(i)}})
	}
	l := sl.List()
	order := make([]*InternalNode, len(l))
	copy(order, l)
	for i := 0; i < 999; i += 1 {
		sl.Next()
	}

	func() {
		l := sl.List()
		for i, v := range order {
			if l[i] != v {
				return
			}
		}
		t.Fatalf("expected list to be shuffled")
	}()

	if sl.Next() == n1 && sl.Next() == n2 {
		t.Fatalf("expected list to be shuffled")
	}
}
