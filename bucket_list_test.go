package swim

import (
	"sort"
	"testing"
)

func TestBucketList(t *testing.T) {
	localNode := &Node{}
	bl := &BucketList{K: 3,
		Sort: func(nodes []*InternalNode, local *Node) error {
			if local != localNode {
				t.Fatalf("Expected local node %v got %v", localNode, local)
			}
			sort.Sort(byId(nodes))
			return nil
		},
		LocalNode: localNode,
	}

	count := 0
	node := func() *InternalNode {
		count += 1
		return &InternalNode{Node: Node{Id: uint64(count)}}
	}

	testBucketLen := func() {
		testListLen(t, bl, count)
		i := uint64(1)
		for _, b := range bl.buckets {
			nodes := b.List()
			sort.Sort(byId(nodes))
			for _, n := range nodes {
				if i > uint64(count) {
					t.Fatalf("expected %v IDs got %v", count, i)
				}
				if n.Id != i {
					t.Fatalf("expected ID %v got %v", i, n.Id)
				}
				i += 1
			}
		}
	}

	testBucketLenths := func(sizes []int) {
		if n, m := len(sizes), len(bl.buckets); n != m {
			t.Fatalf("expected %v buckets got %v", n, m)
		}
		for i, size := range sizes {
			if l := bl.buckets[i].Len(); l != size {
				t.Fatalf("expected bucket of size %v got %v", size, l)
			}
		}
	}

	if n := bl.Next(); n != nil {
		t.Fatalf("expected nil got %v", n)
	}

	// 1 node: 0 0 1
	n1 := node()
	bl.Add(n1)
	testBucketLenths([]int{0, 0, 1})
	testBucketLen()

	if n := bl.Next(); n != n1 {
		t.Fatalf("expected %v got %v", n1, n)
	}

	// 2 nodes: 0 0 2
	bl.Add(node())
	testBucketLenths([]int{0, 0, 2})
	testBucketLen()

	// 3 nodes: 0 1 2
	bl.Add(node())
	testBucketLenths([]int{0, 1, 2})
	testBucketLen()

	// 4 nodes: 0 1 3
	bl.Add(node())
	testBucketLenths([]int{0, 1, 3})
	testBucketLen()

	// 5 nodes: 0 2 3
	bl.Add(node())
	testBucketLenths([]int{0, 2, 3})
	testBucketLen()

	// 6 nodes: 0 2 4
	bl.Add(node())
	testBucketLenths([]int{0, 2, 4})
	testBucketLen()

	// 7 nodes: 1 2 4
	bl.Add(node())
	testBucketLenths([]int{1, 2, 4})
	testBucketLen()

	// 8 nodes: 1 2 5
	bl.Add(node())
	testBucketLenths([]int{1, 2, 5})
	testBucketLen()

	// 15 nodes: 2 4 9
	bl.Add(node(), node(), node(), node(), node(), node(), node())
	testBucketLenths([]int{2, 4, 9})
	testBucketLen()

	// 22 nodes: 3 6 13
	bl.Add(node(), node(), node(), node(), node(), node(), node())
	testBucketLenths([]int{3, 6, 13})
	testBucketLen()
}
