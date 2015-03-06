# Data Structures

This document describes the data structures for `go-swim` and the design rationale for each data structure and method, as it pertains to implementing the SWIM failure detector for rapid experimentation. In designing the data structures, I tried to localize knowledge as much as possible. For example, knowledge about how to contact a node is best encapsulated in the `Node` data structure. On the other hand, knowledge about how to select that node is better kept in a container for the nodes.

The design of `go-swim` borrows heavily from HashiCorp's [`memberlist`][memberlist] implementation of SWIM.


## `Node` as the fundamental unit of information

```go
type Node struct {
    Name       string      // User-provided node name
    Addrs      []*net.Addr // List of addresses assigned to the node
    Meta       []byte      // Metadata from the delegate for this node.
}
```

Internally, SWIM processes are referred to as nodes. The nodes are the basic unit of information in that without them, there would be no use for SWIM. The difference between `Node` and `NodeState` is that the former is returned to end-client applications, while the latter exposes internal properties intended for delegate handlers.

`Name` and `Addrs`: Nodes minimally maintain the user-provided node name and a list of network addresses at which the node may be reached.

While the first iterations of the software may only support using the first address in the list, future revisions should at least be able to contact nodes using more than one network address. This is a likely scenario when nodes are mobile (e.g. cell phones) or when nodes are attached to more than one disjoint networks (e.g. a primary network and a backup network).

We also assume that, for the first iteration of the software, that node names are unique. While `memberlist` rejects new nodes with the same name as an existing node, our (eventual) support for multiple addresses doesn't have as clear of a solution.


### `InternalNode` for internal state information

```go
type NodeState int

const (
    StateAlive NodeStateType = iota
    StateSuspect
    StateDead
)

type InternalNode {
    Node
    InternalId       []byte        // Byte representation of the node name
    OrderingId       []byte        // ID representation used for ordering nodes
    Incarnation      uint32        // Last known incarnation number
    State            NodeState     // Current state
    StateLastUpdated time.Time     // Last time the state was updated
}
```

`InternalId` and `OrderingId`: To better support the Chord and Kademlia distance metrics used in ordering the nodes, we maintain an internal byte representation of the user-provided node name (e.g. a cryptographic hash of the node name) and a numeric representation of the node name.

`Incarnation`: A node's incarnation number is a global monotonically increasing integer. On joining the group, a node's incarnation number is initialized to `0`. It is incremented only when another node suspects it of failure and when a node refutes its failure, with both messages broadcast to the group using the dissemination component.

`State`: The state describes the node's membership status in the group, as required of any failure detector.

`StateLastUpdated`: The last update time for a node is used to detect when a node has not replied to a ping or refuted its suspicion status.


### `NodeVisitor` for populating node fields

```go
type NodeVisitor func(localNode *InternalNode, targetNode *InternalNode) (*InternalNode, error)

// Implementations of NodeVisitor
func HashNodeVisitor(hash func() hash.Hash) NodeVisitor {}
func XorNodeVisitor(localNode *InternalNode, targetNode *InternalNode) (*InternalNode, error) {}
```

The `NodeVisitor` is responsible for updating an `InternalNode` for the given local `InternalNode`, modifying the relevant `InternalNode` fields as necessary. For example, it is used to implement the Chord and Kademlia distance metrics.

`HashNodeVisitor` creates a node visitor that uses the given hash function to set the `InternalId` field of the target `InternalNode`. It is used for generating the "random" IDs needed by the Chord fingers and Kademlia XOR metric.

`XorNodeVisitor` sets the `OrderingId` by XORing the local and target node `InternalIds`.


### `NodeSorter` for sorting nodes

```go
type NodeSorter func(nodes []*InternalNode) error

// Implementations of NodeSorter
func OrderingIdNodeSorter(nodes []*InternalNode) error {}
func ChordFingerNodeSorter(nodes []*InternalNode) error {}
```

The `NodeSorter` reorders the given nodes in ascending order. This is not necessarily the simple comparison-based sort used by `OrderingIdNodeSorter`. The `ChordFingerNodeSorter`, for example, orders the nodes based on the global ordering provided by `OrderingIdNodeSorter`.


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
type NeighborhoodSelectionList struct {
    k           uint                 // Number of regional buckets to maintain
    r           uint                 // Size of the neighborhood
    s           uint                 // Number of nodes to select from the neighborhood
    region      BucketSelectionList  // Regional nodes are more than r nodes away
    neighbors   ShuffleSelectionList // Neighboring nodes are within r nodes distance
    sortedNodes []*InternalNode      // List of nodes sorted by OrderingId
}
```

The `NodeSelectionList` interface defines generic methods for managing a list of nodes for the purpose of choosing candidate nodes for pinging. The `ShuffleSelectionList` implements the round-robin shuffle method described in SWIM. The `BucketSelectionList` implements round-robin shuffle over buckets of sizes `ceil(n*(2/3)*(1/3)^i)` where `n` is the total number of nodes, `0 <= i < k` are the bucket numbers, and `k` is the number of nodes to ping during each protocol period. The `NeighborhoodSelectionList` selects `s` nodes from the `r` closest neighboring nodes and the rest from the regional node list.


## `AftershockTicker` for periodically pinging peers

```go
type AftershockTicker struct {
    Period      time.Duration      // The duration between primary ticks
    PhaseDelays []time.Duration    // The phase delays for aftershock ticks
    C           <-chan time.Time   // The channel on which primary ticks are delivered
    Q           []<-chan time.Time // The channels on which aftershock ticks are delivered
}
```

The `AftershockTicker` implements a ticker that also delivers phase-shifted aftershock ticks. The ticker is used to coalesce related timeouts in the basic SWIM failure detection algorithm. The primary tick triggers the first round of `k` pings, followed by a second aftershock tick shifted by the algorithm timeout to check if any node need to be indirectly pinged. For example:

```
| Primary tick ------------>| Second aftershock tick
| Ping k nodes              | Indirectly ping nodes if no reply received
```


- `MemberList`
    + `Conn *net.PacketConn`
    + `timer time.Interval`
    + `bucketList BucketList`
    + `Ping()`
    + `Start()`
    + `Stop()`


[memberlist]: https://github.com/hashicorp/memberlist
