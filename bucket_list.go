package swim

import (
// "math"
)

// A bucket list selects nodes using round-robin over buckets of nodes. Each
// bucket is at least twice as large as the next smaller bucket. The methods
// are not safe to run concurrently.
type BucketList struct {
	K         uint   // Number of buckets to maintain
	Sort      Sorter // Sorter implementation
	LocalNode *Node

	nodes      []*InternalNode // List of nodes
	buckets    []*ShuffleList  // List of buckets
	nextBucket int
}

// Add nodes to the list.
func (l *BucketList) Add(nodes ...*InternalNode) {
	k := l.K

	// check required properties
	if k < 2 {
		panic("K < 2")
	} else if l.LocalNode == nil {
		panic("Sort == nil")
	} else if l.Sort == nil {
		panic("LocalNode == nil")
	}

	// set next set of nodes
	l.SetNext(append(l.nodes, nodes...))
}

// Remove nodes from the list.
func (l *BucketList) Remove(removes ...*InternalNode) {
	// TODO: make this more efficient
	nodes := make([]*InternalNode, 0, len(l.nodes))
	for _, bucket := range l.buckets {
		bucket.Remove(removes...)
		nodes = append(nodes, bucket.List()...)
	}
	l.SetNext(nodes)
}

// Set the next list of nodes from which to select
func (l *BucketList) SetNext(nodes []*InternalNode) {
	k := int(l.K)

	// copy nodes to prevent modifying the underlying array
	localNodes := make([]*InternalNode, len(nodes))
	copy(localNodes, nodes)

	// update number of buckets
	buckets := l.buckets
	n := len(buckets)
	if n < k {
		for ; n < k; n += 1 {
			buckets = append(buckets, new(ShuffleList))
		}
	} else if n > k {
		buckets = buckets[:k]
	}

	// sort nodes
	l.Sort(localNodes, l.LocalNode)

	// populate buckets
	unallocated := localNodes
	for i := k - 1; i > 0; i -= 1 {
		q := 1 << uint(i)
		d := (q << 1) - 1
		l := len(unallocated)
		n := (l*q + d - 1) / d
		bucket := buckets[i]
		bucket.SetNext(unallocated[l-n:])
		unallocated = unallocated[:l-n]
	}

	// populate first bucket
	bucket := buckets[0]
	bucket.SetNext(unallocated)

	// save changes
	l.nodes = localNodes
	l.buckets = buckets
}

// Select a node from the list.
func (l *BucketList) Next() *InternalNode {

	// edge case
	if len(l.buckets) == 0 {
		return nil
	}

	// get next node
	i := l.nextBucket % len(l.buckets)
	for range l.buckets {
		if node := l.buckets[i].Next(); node != nil {
			l.nextBucket = (i + 1) % len(l.buckets)
			return node
		}
		i = (i + 1) % len(l.buckets)
	}

	// all buckets empty
	return nil
}

// Get a list of the contained nodes. The returned list references the
// internal slice and should not be modified.
func (l *BucketList) List() []*InternalNode {
	return l.nodes
}

// Get the length of the list.
func (l *BucketList) Len() int {
	return len(l.nodes)
}
