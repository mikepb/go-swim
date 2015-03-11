package swim

// IDs are addressed with 64-bits, as in the IEEE EUI-64 format.
type Id [8]byte

// Node describes a member of the group.
type Node struct {
	Id       Id          // Node ID
	Addrs    []Addr      // List of addresses assigned to the node
	UserData interface{} // User data
}

// A node can have one of three states: alive, suspect, or dead.
type State uint8

const (
	_       State = iota
	Alive         // Node is alive
	Suspect       // Node is suspected of failure
	Dead          // Node is confirmed as failed
)

// InternalNode maintains the state of a node in the failure detector.
type InternalNode struct {
	Incarnation Seq // Last known incarnation number
	AckSeq      Seq // Last sequence for which the node has acknowledged
	ContactSeq  Seq // Last sequence in which a message was sent
	RTT         RTT // Round-trip time estimator

	Node
	State     State  // Node membership state
	SortValue uint64 // For the sorting implementations
}
