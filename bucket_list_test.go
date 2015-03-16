package swim

import (
	"sort"
	"testing"
)

func TestBucketList(t *testing.T) {
	bl := &BucketList{K: 3,
		Sort: func(nodes []*InternalNode, localNode *InternalNode) error {
			sort.Sort(byId(nodes))
			return nil
		},
		LocalNode: &InternalNode{},
	}

	count := 0
	node := func() *InternalNode {
		count += 1
		return &InternalNode{Node: Node{Id: uint64(count)}}
	}

	testBucketSize := func() {
		testListSize(t, bl, count)
		i := uint64(1)
		for _, b := range bl.buckets {
			nodes := b.Nodes
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

	if n := bl.Next(); n != nil {
		t.Fatalf("expected nil got %v", n)
	}

	// 1 node: 0 0 1
	n1 := node()
	bl.Add(n1)
	testBucketSizes(t, bl, []int{0, 0, 1})
	testBucketSize()

	if n := bl.Next(); n != n1 {
		t.Fatalf("expected %v got %v", n1, n)
	}

	// 2 nodes: 0 0 2
	bl.Add(node())
	testBucketSizes(t, bl, []int{0, 0, 2})
	testBucketSize()

	// 3 nodes: 0 1 2
	bl.Add(node())
	testBucketSizes(t, bl, []int{0, 1, 2})
	testBucketSize()

	// 4 nodes: 0 1 3
	bl.Add(node())
	testBucketSizes(t, bl, []int{0, 1, 3})
	testBucketSize()

	// 5 nodes: 0 2 3
	bl.Add(node())
	testBucketSizes(t, bl, []int{0, 2, 3})
	testBucketSize()

	// 6 nodes: 0 2 4
	bl.Add(node())
	testBucketSizes(t, bl, []int{0, 2, 4})
	testBucketSize()

	// 7 nodes: 1 2 4
	bl.Add(node())
	testBucketSizes(t, bl, []int{1, 2, 4})
	testBucketSize()

	// 8 nodes: 1 2 5
	bl.Add(node())
	testBucketSizes(t, bl, []int{1, 2, 5})
	testBucketSize()

	// 15 nodes: 2 4 9
	bl.Add(node(), node(), node(), node(), node(), node(), node())
	testBucketSizes(t, bl, []int{2, 4, 9})
	testBucketSize()

	// 22 nodes: 3 6 13
	bl.Add(node(), node(), node(), node(), node(), node(), node())
	testBucketSizes(t, bl, []int{3, 6, 13})
	testBucketSize()
}

func testBucketSizes(t *testing.T, bucket *BucketList, sizes []int) {
	if n, m := len(sizes), len(bucket.buckets); n != m {
		t.Fatalf("expected %v buckets got %v", n, m)
	}
	for i, size := range sizes {
		if l := bucket.buckets[i].Size(); l != size {
			t.Fatalf("expected bucket of size %v got %v", size, l)
		}
	}
}
