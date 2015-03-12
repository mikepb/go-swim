package swim

import (
	"math/rand"
)

// A shuffle list selects nodes round-robin, shuffling the list after each
// round. The methods are not safe to run concurrently.
type ShuffleList struct {
	Nodes     []*InternalNode // List of nodes
	nextIndex int
}

// Add nodes to the end of the list.
func (l *ShuffleList) Add(nodes ...*InternalNode) {
	l.Nodes = append(l.Nodes, nodes...)
}

// Remove nodes from the list.
func (l *ShuffleList) Remove(nodes ...*InternalNode) {
	// TODO: make this more efficient
	size := len(l.Nodes)
	for i := 0; i < size; {
		for _, n := range nodes {
			if l.Nodes[i] == n {
				size -= 1
				l.Nodes[i], l.Nodes[size] = l.Nodes[size], nil
				i += 1
			}
		}
		i += 1
	}
	l.Nodes = l.Nodes[0:size]
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
	for i := len(l.Nodes) - 1; i > 0; i -= 1 {
		j := rand.Intn(i + 1)
		l.Nodes[i], l.Nodes[j] = l.Nodes[j], l.Nodes[i]
	}
}

// Get a list of the contained nodes. The returned list references the
// internal slice and should not be modified.
func (l *ShuffleList) List() []*InternalNode {
	return l.Nodes
}

// Get the size of the list.
func (l *ShuffleList) Size() int {
	return len(l.Nodes)
}
