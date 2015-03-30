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
	} else if s := sl.Len(); s != 2 {
		t.Fatalf("expected 2 nodes, got %v", s)
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

	nodes := make([]*InternalNode, 997)
	for i := range nodes {
		nodes[i] = &InternalNode{Node: Node{Id: uint64(i + 999)}}
	}
	sl.Add(nodes...)
	l := sl.List()
	order := make([]*InternalNode, len(l))
	copy(order, l)

	for i := range nodes {
		if id := sl.Next().Id; id != uint64(i+999) {
			t.Fatalf("expected node %v got %v", i+999, id)
		}
	}

	if s := sl.Len(); s != 999 {
		t.Fatalf("expected 2 nodes, got %v", s)
	}

	next := sl.Next()

	func() {
		l := sl.List()
		for i, v := range order {
			if l[i] != v {
				return
			}
		}
		t.Fatalf("expected list to be shuffled")
	}()

	if next == n1 && sl.Next() == n2 {
		t.Fatalf("expected list to be shuffled")
	}

	sl.Remove(n1, n2)
	if s := sl.Len(); s != 997 {
		t.Fatalf("expected 2 nodes, got %v", s)
	}

	sl.Replace([]*InternalNode{})
	sl.nextIndex = len(sl.nodes)
	if sl.Next() != nil {
		t.Fatalf("expected nil node")
	} else if len(sl.nodes) != 0 {
		t.Fatalf("expected empty list")
	}

	sl.Replace([]*InternalNode{n1})
	if sl.Next() != n1 {
		t.Fatalf("expected node 1")
	} else if l := len(sl.nodes); l != 1 {
		t.Fatalf("expected list of size %v got %v", 1, l)
	}
}
