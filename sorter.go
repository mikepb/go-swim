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
	l := len(nodes)

	// prepare sorted list
	presorted := make([]*InternalNode, l)
	copy(presorted, nodes)
	sort.Sort(byId(presorted))

	// find starting index
	startIndex := sort.Search(l, func(i int) bool {
		return presorted[i].Id >= localNode.Id
	}) % l

	// populate output list by chord fingers
	for i := range nodes {
		// next finger index
		j := (startIndex + (1 << uint(i)) - 1) % l
		// find next available node
		for presorted[j] == nil {
			j = (j + 1) % l
		}
		// save in output list
		nodes[i] = presorted[j]
		presorted[j] = nil
	}

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
