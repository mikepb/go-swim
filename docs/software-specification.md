# Software Specification

This document describes the software architecture for `go-swim` and the design rationale for each data structure and method, as it pertains to implementing the SWIM failure detector for rapid experimentation. In designing the software, I tried to localize knowledge as much as possible. For example, knowledge about how to contact a node is best encapsulated in the `Node` data structure. On the other hand, knowledge about how to select that node is better kept in a container for the nodes.

The design of `go-swim` borrows heavily from HashiCorp's [`memberlist`][memberlist] implementation of SWIM.


## `Node` and `NodeState` as fundamental units of information

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


### `NodeState` for implementation-specific state information

```go
type NodeStateType int

const (
    StateAlive NodeStateType = iota
    StateSuspect
    StateDead
)

type NodeState {
    Node
    InternalId       []byte        // Byte representation of the node name
    OrderingId       *big.Int      // Numeric representation of the internal ID
    Incarnation      uint32        // Last known incarnation number
    State            NodeStateType // Current state
    StateLastUpdated time.Time     // Last time the state was updated
}
```

`InternalId` and `OrderingId`: To better support the Chord and Kademlia distance metrics used in ordering the nodes, we maintain an internal byte representation of the user-provided node name (e.g. a cryptographic hash of the node name) and a numeric representation of the node name.

`Incarnation`: A node's incarnation number is a global monotonically increasing integer. On joining the group, a node's incarnation number is initialized to `0`. It is incremented only when another node suspects it of failure and when a node refutes its failure, with both messages broadcast to the group using the dissemination component.


## `NodeSelectionList` interface and implementations for peer selection

```go
type NodeSelectionList interface {
    func Add(node *Node) error {}
    func Remove(node *Node) error {}
    func Next() (*Node, error) {}
    func List() ([]*Node, error) {}
}

// implements NodeSelectionList
type ShuffleSelectionList struct {
    Nodes     []*Node
    nextIndex int
}

// http://en.wikipedia.org/wiki/Fisher–Yates_shuffle
func (l *ShuffleSelectionList) Shuffle() {}

// implements NodeSelectionList
type BucketSelectionList struct {
    sortedNodes []*Node
    buckets     []*ShuffleSelectionList
}
```

The `NodeSelectionList` interface defines generic methods for managing a list of nodes and choosing candidate nodes for pinging. The `ShuffleSelectionList` implements the round-robin shuffle method described in SWIM. The `BucketSelectionList` implements round-robin shuffle over buckets of sizes `ceil(n*(2/3)*(1/3)^i)` where `n` is the total number of nodes, `0 <= i < k` are the bucket numbers, and `k` is the number of nodes to ping during each protocol period.

- `Delegate`
    + `NodeFactory(name string, addrs []*net.Addr) (*NodeState, error)`

- `MemberList`
    + `Conn *net.PacketConn`
    + `timer time.Interval`
    + `bucketList BucketList`
    + `Ping()`
    + `Start()`
    + `Stop()`


[memberlist]: https://github.com/hashicorp/memberlist
