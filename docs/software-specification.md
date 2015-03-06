# Software Specification

This document describes the software architecture for `go-swim` and the design rationale for each data structure and method, as it pertains to implementing the SWIM failure detector for rapid experimentation. In designing the software, I tried to localize knowledge as much as possible. For example, knowledge about how to contact a node is best encapsulated in the `Node` data structure. On the other hand, knowledge about how to select that node is better kept in a container for the nodes.

The design of `go-swim` borrows heavily from HashiCorp's [`memberlist`][memberlist] implementation of SWIM.


## `Node` the basic unit of information

```go
type Node struct {
    Name       string      // User-provided node name
    Addrs      []*net.Addr // List of addresses assigned to the node
    Meta       []byte      // Metadata from the delegate for this node.
}

type NodeStateType int

const (
    StateAlive NodeStateType = iota
    StateSuspect
    StateDead
)

type NodeState {
    Node
    InternalId   []byte        // Byte representation of the node name
    OrderingId   *big.Int      // Numeric representation of the internal ID
    Incarnation  uint32        // Last known incarnation number
    State        NodeStateType // Current state
    StateUpdated time.Time     // Last time the state was updated
}
```

Internally, SWIM processes are referred to as nodes. The nodes are the basic unit of information in that without them, there would be no use for SWIM.

`Name` and `Addrs`: Nodes minimally maintain the user-provided node name and a list of network addresses at which the node may be reached.

While the first iterations of the software may only support using the first address in the list, future revisions should at least be able to contact nodes using more than one network address. This is a likely scenario when nodes are mobile (e.g. cell phones) or when nodes are attached to more than one disjoint networks (e.g. a primary network and a backup network).

We also assume that, for the first iteration of the software, that node names are unique. While `memberlist` rejects new nodes with the same name as an existing node, our (eventual) support for multiple addresses doesn't have as clear of a solution.

`InternalId` and `OrderingId`: In addition, to better support the Chord and Kademlia distance metrics used in ordering the nodes, we maintain an internal byte representation of the user-provided node name (e.g. a cryptographic hash of the node name) and a numeric representation of the node name.

- `Delegate`
    + `NodeFactory(name string, addrs []*net.Addr) (*NodeState, error)`

- `MemberList`
    + `Conn *net.PacketConn`
    + `timer time.Interval`
    + `bucketList BucketList`
    + `Ping()`
    + `Start()`
    + `Stop()`
- `Bucket`
    + `nodes []*Node`
    + `Next() *Node`
    + `Shuffle()`
    + `Add(*Node)`
    + `Remove(*Node)`
- `BucketList`
    + `nodes []*Node`
    + `Buckets []*Bucket`
    + `Next() *Node`
    + `Add(*Node)`
    + `Remove(*Node)`


[memberlist]: https://github.com/hashicorp/memberlist
