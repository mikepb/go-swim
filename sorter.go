package swim

import (
	"sort"
)

// Sorter sorts nodes relative to a local node according to a distance
// metric.
type Sorter func(nodes []*InternalNode, localNode *Node) error

type byId []*InternalNode

func (s byId) Len() int           { return len(s) }
func (s byId) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byId) Less(i, j int) bool { return s[i].Id < s[j].Id }

type byValue []*InternalNode

func (s byValue) Len() int           { return len(s) }
func (s byValue) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byValue) Less(i, j int) bool { return s[i].SortValue < s[j].SortValue }

// Sort using the Chord finger.
func FingerSorter(nodes []*InternalNode, localNode *Node) error {

	// sort into ring
	if err := RingSorter(nodes, localNode); err != nil {
		return err
	}

	// reset values
	for _, node := range nodes {
		node.SortValue = 0
	}

	// calculate fingers
	max := uint64(len(nodes))
	for i := uint64(1); i-1 < max; i += i {
		nodes[i-1].SortValue = i
	}
	for i, node := range nodes {
		if node.SortValue == 0 {
			node.SortValue = max + uint64(i)
		}
	}

	sort.Sort(byValue(nodes))
	return nil
}

// Sort using successors on a ring.
func RingSorter(nodes []*InternalNode, localNode *Node) error {

	// cache ring position calculation
	for _, node := range nodes {
		if node.SortValue == 0 {
			node.SortValue = uint64(int64(node.Id) - int64(localNode.Id))
		}
	}

	sort.Sort(byValue(nodes))
	return nil
}

// Sort using the Kademlia XOR distance metric.
func XorSorter(nodes []*InternalNode, localNode *Node) (err error) {

	// cache XOR calculation
	for _, node := range nodes {
		if node.SortValue == 0 {
			node.SortValue = localNode.Id ^ node.Id
		}
	}

	sort.Sort(byValue(nodes))
	return nil
}
