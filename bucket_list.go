package swim

import (
// "math"
)

// A bucket list selects nodes using round-robin over buckets of nodes. Each
// bucket is at least twice as large as the next smaller bucket. The methods
// are not safe to run concurrently.
type BucketList struct {
	K         int    // Number of buckets to maintain
	Sort      Sorter // Sorter implementation
	LocalNode *InternalNode

	nodes      []*InternalNode // List of nodes
	buckets    []ShuffleList   // List of buckets
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

	// update buckets
	l.Reset(append(l.nodes, nodes...))
	// TODO: resetting leaves us vulnerable to denial-of-service from
	// malicious nodes broadcasting the presence of non-existent nodes
}

// Remove nodes from the list.
func (l *BucketList) Remove(removes ...*InternalNode) {
	// TODO: make this more efficient
	nodes := make([]*InternalNode, 0, len(l.nodes))
	for _, bucket := range l.buckets {
		bucket.Remove(removes...)
		nodes = append(nodes, bucket.List()...)
	}
	l.Reset(nodes)
}

// Reset the state of the selection list with the given nodes.
func (l *BucketList) Reset(nodes []*InternalNode) {
	k := l.K

	// sort nodes
	l.Sort(nodes, l.LocalNode)

	// create buckets
	buckets := make([]ShuffleList, k)
	localNodes := make([]*InternalNode, len(nodes))
	copy(localNodes, nodes)

	// populate buckets
	unallocated := localNodes
	for i := k - 1; i > 0; i -= 1 {
		q := 1 << uint(i)
		d := (q << 1) - 1
		l := len(unallocated)
		n := (l*q + d - 1) / d
		bucket := &buckets[i]
		bucket.Nodes = unallocated[l-n:]
		bucket.Shuffle()
		unallocated = unallocated[:l-n]
	}

	// populate first bucket
	bucket := &buckets[0]
	bucket.Nodes = unallocated
	bucket.Shuffle()

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

// Get the size of the list.
func (l *BucketList) Size() int {
	return len(l.nodes)
}
