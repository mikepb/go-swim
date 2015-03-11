package swim

// A bucket list selects nodes using round-robin over buckets of nodes. Each
// bucket is at least twice as large as the next smaller bucket.
type BucketList struct {
	N           uint            // Number of buckets to maintain
	SortedNodes []*InternalNode // List of sorted nodes
	Buckets     []ShuffleList   // List of buckets
	Sort        Sorter          // Sorter implementation

	nodes  []*InternalNode // List of unsorted nodes for buckets
	bounds []int           // List of bucket boundaries
}

/*
// Replace the internal list of sorted nodes, updating buckets as needed.
// Primarily used to avoid double-sorting with NeighborhoodSelectionList.
func (l *BucketSelectionList) UpdateInPlace(sortedNodes []*InternalNode) error {}

// Implements NodeSelectionList
type PrioritySelectionList struct {
  k           uint                 // Number of regional buckets to maintain
  r           uint                 // Size of the neighborhood
  s           uint                 // Number of nodes to select from the neighborhood
  region      BucketSelectionList  // Regional nodes are more than r nodes away
  neighbors   ShuffleSelectionList // Neighboring nodes are within r nodes distance
  sortedNodes []*InternalNode      // List of nodes sorted by OrderingId
}
*/
