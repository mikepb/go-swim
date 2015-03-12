package swim

import (
	"math/rand"
)

// A shuffle list selects nodes round-robin, shuffling the list after each
// round. The methods are not safe to run concurrently.
type ShuffleList struct {
	nodes     []*InternalNode // List of nodes
	nextIndex int
}

// Make a new shuffle list a copy of the given node list.
func NewShuffleList(nodes []*InternalNode) *ShuffleList {
	ns := make([]*InternalNode, len(nodes))
	copy(nodes, ns)
	sl := &ShuffleList{nodes: ns}
	sl.Shuffle()
	return sl
}

// Add nodes to the end of the list.
func (l *ShuffleList) Add(nodes ...*InternalNode) {
	l.nodes = append(l.nodes, nodes...)
}

// Remove nodes from the list.
func (l *ShuffleList) Remove(nodes ...*InternalNode) {
	for i, node := range l.nodes {
		for _, n := range nodes {
			if node == n {
				l.nodes[i] = nil
			}
		}
	}
}

// Select a node from the list.
func (l *ShuffleList) Next() *InternalNode {
	i := l.nextIndex
	size := len(l.nodes)

	// empty case
	if size == 0 {
		return nil
	}

	// one node
	if size == 1 {
		return l.nodes[0]
	}

	// shuffle the list if past end
	if i >= size {
		// find last non-nil index
		for size > 0 && l.nodes[size-1] == nil {
			size -= 1
		}
		// remove nil nodes
		for i := 0; i < size; i += 1 {
			if l.nodes[i] == nil {
				l.nodes[i-1] = l.nodes[size]
				size -= 1
				// ignore nil nodes
				for size > 0 && l.nodes[size-1] == nil {
					size -= 1
				}
			}
		}
		// update slice
		l.nodes = l.nodes[0:size]
		// shuffle list
		l.Shuffle()
		i = 0
	}

	// find next non-nil node
	var node *InternalNode
	for node == nil && i < size {
		node = l.nodes[i]
		i += 1
	}

	// update index
	l.nextIndex = i

	// return the node
	return node
}

// Shuffle the list.
func (l *ShuffleList) Shuffle() {
	for i := len(l.nodes) - 1; i > 0; i -= 1 {
		j := rand.Intn(i)
		l.nodes[i], l.nodes[j] = l.nodes[j], l.nodes[j]
	}
}

// Get a list of the contained nodes. The returned list references the
// internal slice and should not be modified.
func (l *ShuffleList) List() []*InternalNode {
	return l.nodes
}
