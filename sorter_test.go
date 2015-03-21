package swim

import (
	"testing"
)

func TestFingerSorter(t *testing.T) {
	nodes, localNode := makeTestNodes()
	if err := FingerSorter(nodes, localNode); err != nil {
		t.Fatal(err)
	}

	// 9 1 2 3 4 5 6

	// _ 1 2 3 4 5 6
	// 9

	// _ _ 2 3 4 5 6
	// 9 1

	// _ _ 2 _ 4 5 6
	// 9 1 3

	// _ _ _ _ 4 5 6
	// 9 1 3 2

	// _ _ _ _ _ 5 6
	// 9 1 3 2 4

	// _ _ _ _ _ _ 6
	// 9 1 3 2 4 5

	// _ _ _ _ _ _ _
	// 9 1 3 2 4 5 6

	expect := []uint64{9, 1, 3, 2, 4, 5, 6}
	actual := make([]uint64, len(expect))
	for i, v := range nodes {
		actual[i] = v.Id
		if v.Id != expect[i] {
			t.Fatalf("expected order %v got %v", expect, actual)
		}
	}
}

func TestXorSorter(t *testing.T) {
	nodes, localNode := makeTestNodes()
	if err := XorSorter(nodes, localNode); err != nil {
		t.Fatal(err)
	}

	// 1  2  3  4  5  6 9
	// 9 10 11 12 13 14 1

	// 9 1  2  3  4  5  6
	// 1 9 10 11 12 13 14

	expect := []uint64{9, 1, 2, 3, 4, 5, 6}
	actual := make([]uint64, len(expect))
	for i, v := range nodes {
		actual[i] = v.Id
		if v.Id != expect[i] {
			t.Fatalf("expected order %v got %v", expect, actual)
		}
	}
}

func makeTestNodes() ([]*InternalNode, *Node) {
	localNode := &Node{Id: 8}
	nodes := []*InternalNode{
		&InternalNode{Node: Node{Id: 6}},
		&InternalNode{Node: Node{Id: 2}},
		&InternalNode{Node: Node{Id: 3}},
		&InternalNode{Node: Node{Id: 9}},
		&InternalNode{Node: Node{Id: 1}},
		&InternalNode{Node: Node{Id: 5}},
		&InternalNode{Node: Node{Id: 4}},
	}
	return nodes, localNode
}
