package swim

import (
	"math/rand"
)

// A shuffle list selects nodes round-robin, shuffling the list after each
// round. The methods are not safe to run concurrently.
type ShuffleList struct {
	Nodes     []*InternalNode // List of nodes
	NextNodes []*InternalNode // List of nodes for the next round
	nextIndex int
}

// Add nodes to the end of the list.
func (l *ShuffleList) Add(nodes ...*InternalNode) {
	l.Nodes = append(l.Nodes, nodes...)
}

// Remove nodes from the list.
func (l *ShuffleList) Remove(nodes ...*InternalNode) {
	// TODO: make this more efficient
	end := len(l.Nodes)
	start := 0

	// for each node in the shuffle list
	for i := 0; i < end; i += 1 {

		// check if it matches any of the nodes to be removed
		for _, n := range nodes {
			if l.Nodes[i] == n {

				if i < l.nextIndex {
					// if we've already visited this node, swap with first node
					l.Nodes[i] = l.Nodes[start]
					start += 1
				} else {
					// otherwise, swap with last
					end -= 1
					l.Nodes[i] = l.Nodes[end]
				}
			}
		}
	}

	// trim unused slots
	l.Nodes = l.Nodes[start:end]
}

// Select a node from the list.
func (l *ShuffleList) Next() *InternalNode {
	i := l.nextIndex
	size := len(l.Nodes)

	// shuffle the list if past end
	if i >= size {

		// swap out next set of nodes
		if l.NextNodes != nil {
			l.Nodes, l.NextNodes = l.NextNodes, nil
			size = len(l.Nodes)
		}

		// randomize the list
		if size > 1 {
			l.Shuffle()
		}

		// reset the next index
		i = 0
	}

	// empty case
	if size == 0 {
		return nil
	}

	// one node
	if size == 1 {
		return l.Nodes[0]
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

// Set the list of nodes to use for the next round.
func (l *ShuffleList) SetNext(nodes []*InternalNode) {
	l.NextNodes = nodes
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
