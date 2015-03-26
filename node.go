package swim

import (
	"time"
)

// A node can have one of three states: alive, suspect, or dead.
type State uint8

const (
	_       State = iota
	Alive         // Node is alive
	Suspect       // Node is suspected of failure
	Dead          // Node is confirmed as failed
)

// Node describes a member of the group.
type Node struct {
	Id          uint64      // Big-endian 64-bit ID, e.g. IEEE EUI-64 format
	Addrs       []string    // Addresses at which to reach the node
	State       State       // Node membership state
	Incarnation Seq         // Last known incarnation number
	UserData    interface{} // User data
}

// InternalNode maintains the state of a node in the failure detector.
type InternalNode struct {
	RTT               RTT       // Round-trip time estimator
	RemoteIncarnation Seq       // Incarnation number of the local node at this node
	LastAckTime       time.Time // Last time the node acknowledged a ping
	LastSentTime      time.Time // Last time a message was sent

	Node
	SortValue uint64 // For the sorting implementations
}
