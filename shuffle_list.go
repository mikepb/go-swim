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
	if l.NextNodes != nil {
		l.NextNodes = append(l.NextNodes, nodes...)
	} else {
		l.Nodes = append(l.Nodes, nodes...)
	}
}

// Remove nodes from the list.
func (l *ShuffleList) Remove(nodes ...*InternalNode) {

	// make id map of nodes to remove
	rms := make(map[uint64]bool)
	for _, node := range nodes {
		rms[node.Id] = true
	}

	// remove from current list
	l.Nodes = l.remove(l.Nodes, rms, l.nextIndex)

	// remove from next list
	if l.NextNodes != nil {
		l.NextNodes = l.remove(l.NextNodes, rms, 0)
	}
}

func (s *ShuffleList) remove(nodes []*InternalNode, removes map[uint64]bool, nextIndex int) []*InternalNode {
	end := len(nodes)
	start := 0

	// for each node in the shuffle list
	for i := 0; i < end; i += 1 {
		node := nodes[i]

		// check if it matches any of the nodes to be removed
		if !removes[node.Id] {
			continue
		}

		if i < nextIndex {
			// if we've already visited this node, swap with first node
			nodes[i] = nodes[start]
			start += 1
		} else {
			// otherwise, swap with last
			end -= 1
			nodes[i] = nodes[end]
		}
	}

	// trim unused slots
	return nodes[start:end]
}

// Select a node from the list.
func (l *ShuffleList) Next() *InternalNode {
	i := l.nextIndex
	size := len(l.Nodes)

	// shuffle the list if past end
	if i >= size {
		l.Shuffle()
		size = len(l.Nodes)
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

	// get next node
	var node *InternalNode
	if i < size {
		node = l.Nodes[i]
	}

	// update index
	l.nextIndex = i + 1

	// return the node
	return node
}

// Set the list of nodes to use for the next round. This method is used to
// support efficient node updates from a bucket list.
func (l *ShuffleList) SetNext(nodes []*InternalNode) {
	if len(l.Nodes) == 0 {
		l.Nodes, l.NextNodes = nodes, nil
		l.Shuffle()
	} else {
		l.NextNodes = nodes
	}
}

// Shuffle the list.
func (l *ShuffleList) Shuffle() {

	// swap out next set of nodes
	if l.NextNodes != nil {
		l.Nodes, l.NextNodes = l.NextNodes, nil
	}

	// randomize list
	for i := len(l.Nodes) - 1; i > 0; i -= 1 {
		j := rand.Intn(i + 1)
		l.Nodes[i], l.Nodes[j] = l.Nodes[j], l.Nodes[i]
	}
}

// Get a list of the contained nodes. The returned list references the
// internal slice and should not be modified. If a list of nodes to use for
// the next round is set, that list is returned instead of the currently
// active list.
func (l *ShuffleList) List() []*InternalNode {
	if l.NextNodes != nil {
		return l.NextNodes
	}
	return l.Nodes
}

// Get the length of the list. If a list of nodes to use for the next round
// is set, the length of that list is returned instead of the length of the
// currently active list.
func (l *ShuffleList) Len() int {
	if l.NextNodes != nil {
		return len(l.NextNodes)
	}
	return len(l.Nodes)
}
