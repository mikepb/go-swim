# Data Structures

This document describes the data structures for `go-swim` and the design rationale for each data structure and method, as it pertains to implementing the SWIM failure detector for rapid experimentation. In designing the data structures, I tried to localize knowledge as much as possible. For example, knowledge about how to contact a node is best encapsulated in the `Node` data structure. On the other hand, knowledge about how to select that node is better kept in a container for the nodes.

The design of `go-swim` borrows heavily from HashiCorp's [`memberlist`][memberlist] implementation of SWIM.


## `Node` as the fundamental unit of information

```go
type Node struct {
    Id    []byte      // Node ID
    Addrs []*net.Addr // List of addresses assigned to the node
    Meta  []byte      // Metadata from the delegate for this node.
}
```

Internally, SWIM processes are referred to as nodes. The nodes are the basic unit of information in that without them, there would be no use for SWIM. The difference between `Node` and `NodeState` is that the former is returned to end-client applications, while the latter exposes internal properties intended for delegate handlers.

`Id` and `Addrs`: Nodes minimally maintain the user-provided node ID and a list of network addresses at which the node may be reached. All node IDs must have the same byte length.

While the first iterations of the software may only support using the first address in the list, future revisions should at least be able to contact nodes using more than one network address. This is a likely scenario when nodes are mobile (e.g. cell phones) or when nodes are attached to more than one disjoint networks (e.g. a primary network and a backup network).

We also assume that, for the first iteration of the software, that node IDs are unique. While `memberlist` rejects new nodes with the same name as an existing node, our (eventual) support for multiple addresses doesn't have as clear of a solution.


### `InternalNode` for internal state information

```go
type NodeState int

const (
    NodeAlive NodeState = iota
    NodeSuspect
    NodeDead
)

type InternalNode {
    Node
    SortValue        interface{}   // For the sorting implementation
    Incarnation      uint32        // Last known incarnation number
    State            NodeState     // Current state
    StateLastUpdated time.Time     // Last time the state was updated
}
```

`SortValue`: The `SortValue` field is used to cache the Kademlia XOR metric. Other sorting implementations may use this field as needed.

`Incarnation`: A node's incarnation number is a global monotonically increasing integer. On joining the group, a node's incarnation number is initialized to `0`. It is incremented only when another node suspects it of failure and when a node refutes its failure, with both messages broadcast to the group using the dissemination component.

`State`: The state describes the node's membership status in the group, as required of any failure detector.

`StateLastUpdated`: The last update time for a node is used to detect when a node has not replied to a ping or refuted its suspicion status.


### `NodeSorter` for sorting nodes

```go
type NodeSorter func(nodes []*InternalNode) error

// Implementations of NodeSorter
func ChordFingerNodeSorter(nodes []*InternalNode) error {}
func XorNodeSorter(nodes []*InternalNode) error {}
```

The `NodeSorter` reorders the given nodes in ascending order according to an implementation-specific distance metric. The sorting is not necessarily comparison-based. The `ChordFingerNodeSorter`, for example, orders the nodes based on the global lexicographical ordering of the node IDs. The `XorNodeSorter`, on the other hand, uses comparison-based sorting over the exclusive or (XOR) of the local node ID and the target node IDs. The local node is guaranteed to be the first node in the given list and must remain in that position.


## `NodeSelectionList` for peer selection

```go
type NodeSelectionList interface {
    func Add(...nodes []*InternalNode) error {}
    func Remove(...nodes []*InternalNode) error {}
    func Choose(nodes []*InternalNode) ([]*InternalNode, error) {}
    func Next() (*InternalNode, error) {}
    func List() ([]*InternalNode, error) {}
}

// Implements NodeSelectionList
type ShuffleSelectionList struct {
    Nodes     []*InternalNode
    nextIndex int
}

// http://en.wikipedia.org/wiki/Fisher–Yates_shuffle
func (l *ShuffleSelectionList) shuffle() {}

// Implements NodeSelectionList
type BucketSelectionList struct {
    k           uint                    // Number of buckets to maintain
    sortedNodes []*InternalNode         // List of nodes sorted by OrderingId
    buckets     []*ShuffleSelectionList // List of buckets
}

// Replace the internal list of sorted nodes, updating buckets as needed.
// Primarily used to avoid double-sorting with NeighborhoodSelectionList.
func (l *BucketSelectionList) UpdateInPlace(sortedNodes []*InternalNode) error {}

// Implements NodeSelectionList
type PrioritySelectionList struct {
    k           uint                 // Number of regional buckets to maintain
    r           uint                 // Size of the neighborhood
    s           uint                 // Number of nodes to select from the neighborhood
    region      BucketSelectionList  // Regional nodes are more than r nodes away
    neighbors   ShuffleSelectionList // Neighboring nodes are within r nodes distance
    sortedNodes []*InternalNode      // List of nodes sorted by OrderingId
}
```

The `NodeSelectionList` interface defines generic methods for managing a list of nodes for the purpose of choosing candidate nodes for pinging. The `ShuffleSelectionList` implements the round-robin shuffle method described in SWIM. The `BucketSelectionList` implements round-robin shuffle over buckets of sizes `ceil(n*(2/3)*(1/3)^i)` where `n` is the total number of nodes, `0 <= i < p` are the bucket numbers, and `p` is the number of nodes to ping during each protocol period. The `PrioritySelectionList` selects `s` nodes from the `r` closest neighboring nodes and the rest from the regional node list.


## `AftershockTicker` for periodically pinging peers

```go
type AftershockTicker struct {
    Period      time.Duration      // The duration between primary ticks
    PhaseDelays []time.Duration    // The phase delays for aftershock ticks
    C           <-chan time.Time   // The channel on which primary ticks are delivered
    Q           []<-chan time.Time // The channels on which aftershock ticks are delivered
}
```

The `AftershockTicker` implements a ticker that also delivers phase-shifted aftershock ticks. The ticker is used to coalesce related timeouts in the basic SWIM failure detection algorithm. The primary tick triggers the first round of `p` pings, followed by a second aftershock tick shifted by the algorithm timeout to check if any node need to be indirectly pinged. For example:

```
| Primary tick ------------>| Second aftershock tick
| Ping `p` nodes            | Use `k` nodes for indirect pings, if needed
```


- `MemberList`
    + `Conn *net.PacketConn`
    + `timer time.Interval`
    + `bucketList BucketList`
    + `Ping()`
    + `Start()`
    + `Stop()`


[memberlist]: https://github.com/hashicorp/memberlist
