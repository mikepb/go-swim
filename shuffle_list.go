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

// Add nodes to the end of the list.
func (l *ShuffleList) Add(nodes ...*InternalNode) {
	l.nodes = append(l.nodes, nodes...)
}

// Remove nodes from the list.
func (l *ShuffleList) Remove(nodes ...*InternalNode) {

	// make id map of nodes to remove
	rms := make(map[uint64]bool)
	for _, node := range nodes {
		rms[node.Id] = true
	}

	// remove from current list
	l.nodes = l.remove(l.nodes, rms, l.nextIndex)
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
	size := len(l.nodes)

	// shuffle the list if past end
	if i >= size {
		l.Shuffle()
		size = len(l.nodes)
		i = 0
	}

	// empty case
	if size == 0 {
		return nil
	}

	// one node
	if size == 1 {
		return l.nodes[0]
	}

	// get next node
	var node *InternalNode
	if i < size {
		node = l.nodes[i]
	}

	// update index
	l.nextIndex = i + 1

	// return the node
	return node
}

// Set the list of nodes to use for the next round.
func (l *ShuffleList) Replace(nodes []*InternalNode) {
	l.nodes = nodes
	l.nextIndex = 0
	l.Shuffle()
}

// Shuffle the list.
func (l *ShuffleList) Shuffle() {
	for i := len(l.nodes) - 1; i > 0; i -= 1 {
		j := rand.Intn(i + 1)
		l.nodes[i], l.nodes[j] = l.nodes[j], l.nodes[i]
	}
}

// Get a list of the contained nodes. The returned list references the
// internal slice and should not be modified.
func (l *ShuffleList) List() []*InternalNode {
	return l.nodes
}

// Get the length of the list.
func (l *ShuffleList) Len() int {
	return len(l.nodes)
}
