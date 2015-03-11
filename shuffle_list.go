package swim

import (
	"math/rand"
)

// A shuffle list selects nodes round-robin, shuffling the list after each
// round. The methods are not safe to run concurrently.
type ShuffleList struct {
	Nodes     []*InternalNode // List of nodes
	Sort      Sorter          // Sorter implementation
	nextIndex int
}

// Add nodes to the end of the list.
func (l *ShuffleList) Add(nodes ...*InternalNode) {
	l.Nodes = append(l.Nodes, nodes...)
}

// Remove nodes from the list.
func (l *ShuffleList) Remove(nodes ...*InternalNode) {
	for i, node := range l.Nodes {
		for _, n := range nodes {
			if node == n {
				l.Nodes[i] = nil
			}
		}
	}
}

// Select a node from the list.
func (l *ShuffleList) Next() *InternalNode {
	i := l.nextIndex
	size := len(l.Nodes)

	// empty case
	if size == 0 {
		return nil
	}

	// one node
	if size == 1 {
		return l.Nodes[0]
	}

	// shuffle the list if past end
	if i >= size {
		// find last non-nil index
		for size > 0 && l.Nodes[size-1] == nil {
			size -= 1
		}
		// remove nil nodes
		for i := 0; i < size; i += 1 {
			if l.Nodes[i] == nil {
				l.Nodes[i-1] = l.Nodes[size]
				size -= 1
				// ignore nil nodes
				for size > 0 && l.Nodes[size-1] == nil {
					size -= 1
				}
			}
		}
		// update slice
		l.Nodes = l.Nodes[0:size]
		// shuffle list
		l.Shuffle()
		i = 0
	}

	// find next non-nil node
	var node *InternalNode
	for node == nil && i < size {
		node = l.Nodes[i]
		i += 1
	}

	// update index
	l.nextIndex = i

	// return the node
	return node
}

// Shuffle the list.
func (l *ShuffleList) Shuffle() {
	for i := len(l.Nodes); i > 0; i -= 1 {
		j := rand.Intn(i)
		l.Nodes[i], l.Nodes[j] = l.Nodes[j], l.Nodes[j]
	}
}

// Get a list of the contained nodes. The returned list references the
// internal slice and should not be modified.
func (l *ShuffleList) List() []*InternalNode {
	return l.Nodes
}
