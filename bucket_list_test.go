package swim

import (
	"testing"
)

func TestBucketList(t *testing.T) {
	bl := &BucketList{K: 3,
		Sort: func(nodes []*InternalNode, localNode *InternalNode) error {
			return nil
		},
		LocalNode: &InternalNode{},
	}

	if n := bl.Next(); n != nil {
		t.Fatalf("expected nil got %v", n)
	}

	// 1 node: 0 0 1
	n1 := &InternalNode{}
	bl.Add(n1)
	testBucketSizes(t, bl, []int{0, 0, 1})
	testListSize(t, bl, 1)

	if n := bl.Next(); n != n1 {
		t.Fatalf("expected %v got %v", n1, n)
	}

	// 2 nodes: 0 0 2
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{0, 0, 2})
	testListSize(t, bl, 2)

	// 3 nodes: 0 1 2
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{0, 1, 2})
	testListSize(t, bl, 3)

	// 4 nodes: 0 1 3
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{0, 1, 3})
	testListSize(t, bl, 4)

	// 5 nodes: 0 2 3
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{0, 2, 3})
	testListSize(t, bl, 5)

	// 6 nodes: 0 2 4
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{0, 2, 4})
	testListSize(t, bl, 6)

	// 7 nodes: 1 2 4
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{1, 2, 4})
	testListSize(t, bl, 7)

	// 8 nodes: 1 2 5
	bl.Add(&InternalNode{})
	testBucketSizes(t, bl, []int{1, 2, 5})
	testListSize(t, bl, 8)

	// 15 nodes: 2 4 9
	bl.Add(&InternalNode{}, &InternalNode{}, &InternalNode{}, &InternalNode{})
	bl.Add(&InternalNode{}, &InternalNode{}, &InternalNode{})
	testBucketSizes(t, bl, []int{2, 4, 9})
	testListSize(t, bl, 15)

	// 22 nodes: 3 6 13
	bl.Add(&InternalNode{}, &InternalNode{}, &InternalNode{}, &InternalNode{})
	bl.Add(&InternalNode{}, &InternalNode{}, &InternalNode{})
	testBucketSizes(t, bl, []int{3, 6, 13})
	testListSize(t, bl, 22)
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
