package swim

import (
	"bytes"
	"encoding/binary"
	"sort"
)

// Sorter sorts nodes relative to a local node according to a distance
// metric.
type Sorter func(nodes []*InternalNode, localNode *InternalNode) error

type byValue []*InternalNode

func (s byValue) Len() int           { return len(s) }
func (s byValue) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byValue) Less(i, j int) bool { return s[i].SortValue < s[j].SortValue }

// Sort using the Chord finger.
func FingerSorter(nodes []*InternalNode, localNode *InternalNode) error {
	l := len(nodes)

	// convert ID's to uint64
	for _, node := range nodes {
		if node.SortValue == 0 {
			id, err := idToUint64(&node.Id)
			if err != nil {
				return err
			}
			node.SortValue = id
		}
	}

	// prepare sorted list
	presorted := make([]*InternalNode, l)
	copy(presorted, nodes)
	sort.Sort(byValue(presorted))

	// find starting index
	id := localNode.Id[:]
	startIndex := sort.Search(l, func(i int) bool {
		return bytes.Compare(presorted[i].Id[:], id) >= 0
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
func XorSorter(nodes []*InternalNode, localNode *InternalNode) (err error) {
	var localId uint64

	// cache XOR calculation
	for _, node := range nodes {
		if node.SortValue == 0 {

			// convert local ID to uint64
			if localId == 0 {
				if localId, err = idToUint64(&localNode.Id); err != nil {
					return
				}
			}

			// save XOR calculation
			if id, err := idToUint64(&node.Id); err != nil {
				return err
			} else {
				node.SortValue = id ^ localId
			}
		}
	}

	sort.Sort(byValue(nodes))
	return nil
}

func idToUint64(id *Id) (uid uint64, err error) {
	r := bytes.NewReader(id[:])
	err = binary.Read(r, binary.BigEndian, &uid)
	return
}
