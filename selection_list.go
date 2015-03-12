package swim

// A selection list defines the strategy with which the failure detector
// selects nodes for probing.
type SelectionList interface {
	Add(nodes ...*InternalNode)
	Remove(nodes ...*InternalNode)
	Next() *InternalNode
	List() []*InternalNode
	Size() int
}
