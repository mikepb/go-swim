# Data Structures

This document describes the data structures for `go-swim` and the design rationale for each data structure and method, as it pertains to implementing the SWIM failure detector for rapid experimentation. In designing the data structures, I tried to localize knowledge as much as possible. For example, knowledge about how to contact a node is best encapsulated in the `Node` data structure. On the other hand, knowledge about how to select that node is better kept in a container for the nodes.

The design of `go-swim` borrows heavily from HashiCorp's [`memberlist`][memberlist] implementation of SWIM.


## `Node` as the fundamental unit of information

```go
type Id []byte

type Node struct {
    Id    Id          // Node ID
    Addrs []string    // List of addresses assigned to the node
    Meta  interface{} // Metadata
}
```

Internally, SWIM processes are referred to as nodes. The nodes are the basic unit of information in that without them, there would be no use for SWIM. The difference between `Node` and `NodeState` is that the former is returned to end-client applications, while the latter exposes internal properties intended for delegate handlers.

`Id` and `Addrs`: Nodes minimally maintain the user-provided node ID and a list of network addresses at which the node may be reached. All node IDs must have the same byte length.

While the first iterations of the software may only support using the first address in the list, future revisions should at least be able to contact nodes using more than one network address. This is a likely scenario when nodes are mobile (e.g. cell phones) or when nodes are attached to more than one disjoint networks (e.g. a primary network and a backup network).

We also assume that, for the first iteration of the software, that node IDs are unique. While `memberlist` rejects new nodes with the same name as an existing node, our (eventual) support for multiple addresses doesn't have as clear of a solution.


### `InternalNode` for internal state information

```go
type NodeState uint8

const (
    NodeAlive NodeState = iota
    NodeSuspect
    NodeDead
)

type InternalNode {
    Node
    Incarnation      uint32        // Last known incarnation number
    LastSequenceSeen uint32        // Last sequence in which a message was received
    LastSequenceSent uint32        // Last sequence in which a message was sent
    State            NodeState     // Current state
    SortValue        interface{}   // For the sorting implementation
}
```

`Incarnation`: A node's incarnation number is a global monotonically increasing integer. On joining the group, a node's incarnation number is initialized to `0`. It is incremented only when another node suspects it of failure and when a node refutes its failure, with both messages broadcast to the group using the dissemination component.

`LastSequenceSeen`: The last seen sequence number corresponds to the last protocol period in which a message from the node was received. It is used to detect when a node has not replied to a ping or refuted its suspicion status.

`LastSequenceSent`: The last sent sequence number corresponds to the last protocol period in which a message was sent to the node. This is used to limit the number of messages sent to any given node within a protocol period.

`State`: The state describes the node's membership status in the group, as required of any failure detector.

`SortValue`: The `SortValue` field is used to cache the Kademlia XOR metric. Other sorting implementations may use this field as needed.


### `NodeSorter` for sorting nodes

```go
type NodeSorter func(nodes []*InternalNode, localNode *InternalNode) error

// Implementations of NodeSorter
func ChordFingerNodeSorter(nodes []*InternalNode, localNode *InternalNode) error {}
func XorNodeSorter(nodes []*InternalNode, localNode *InternalNode) error {}
```

The `NodeSorter` reorders the given nodes in ascending order according to an implementation-specific distance metric. The sorting is not necessarily comparison-based. The `ChordFingerNodeSorter`, for example, orders the nodes based on the global lexicographical ordering of the node IDs. The `XorNodeSorter`, on the other hand, uses comparison-based sorting over the exclusive or (XOR) of the local node ID and the target node IDs.


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


## `Transport` for sending and receiving messages

```go
type OutgoingPacket {
    To       *Node
    Messages []interface{}
}

type IncomingPacket {
    From     Id
    Messages []interface{}
}

type MarshalledMessage struct {
    Message interface{}
    Data    []byte
    Size    int
}

type Transport interface {
    Marshal(message []interface{}) (MarshalledMessage, error)
    MaxPacketSize() int
    Outbox() chan-> OutgoingPacket
    Inbox() <-chan IncomingPacket
}
```

`Transport` is responsible for sending messages to other nodes and maintaining an inbox of messages received from other nodes. This object is meant to separate the transport and control panes (https://github.com/hashicorp/memberlist/issues/21) and to ease the implementation of an in-process network simulator. The recognized message structures are described in the next section. Unrecognized messages are passed to the delegate, or the program will panic if no delegate is configured.

The `Marshal()` and `MaxPacketSize()` methods are used to limit the size of transmitted packets, especially useful when attaching broadcast messages. The purpose of the `MarshalledMessage` structure is to cache marshalling operations on messages used to calculate the message size for limiting the size of transmitted packets. It is an optimization; the representation sent to other nodes is identical to the original message.


## `Message` for describing network messages

```go
type MessageHeader struct {
    From  Id
    Stamp uint32 // Message sequence or incarnation
}

type PingMessage struct {
    MessageHeader
    To Id
}

type ProbeMessage struct {
    Ping
    Addrs []string
}

type AckMessage struct {
    MessageHeader
}

type AliveMessage struct {
    MessageHeader
    Node
}

type SuspectMessage struct {
    MessageHeader
    Id Id
}

type DeadMessage struct {
    MessageHeader
    Id Id
}

type UserMessage struct {
    MessageHeader
    Id   Id
    Data interface{}
}
```

These structures describe the messages sent over the transport between peers. `Header` describes the message's sender and a numerical `Stamp` that is interpreted either as the `Sequence` number of the originating node (`Ping`, `Probe`, and `Ack`) or as the `Incarnation` number of the target node (`Alive`, `Suspect`, `Dead`, and `Meta`).

The `Ping` message is sent to probe a node's status. The `Probe` message is sent to third-party nodes to indirectly probe an unresponsive node. The `Ack` message is returned by a directly probed node as well as the intermediate node serving an indirect probe.

Multiple messages are bundled together in a packet and sent as a single addressed unit. See the previous section on the `Transport` interface for more details.


## `BroadcastQueue` for piggybacking broadcasts

```go
type Broadcast interface {
    // Determine if another broadcast invalidates this one.
    Invalidates(b Broadcast) bool

    // Get the message data.
    Message() interface{}

    // Get an estimate of the message size.
    Size() int

    // Invoked when the broadcast reaches its retransmission limit or is
    // invalidated by another broadcast.
    Done()
}

type limitedBroadcast {
    transmits int       // Number of transmissions attempted
    b         Broadcast // The broadcast message
}

// Implements the sort.Interface methods
type priorityBroadcasts []*limitedBroadcast

type BroadcastQueue struct {
    Limit int // Maximum number of queued broadcasts before blocking
    queue priorityBroadcasts
}

// Push a broadcast onto the queue, blocking if Limit > 0 and the queue is
// full.
func (q *BroadcastQueue) Push(b Broadcast) {}

// Match broadcasts up to the given byte size and broadcast message
// transmission limit.
func (q *BroadcastQueue) Match(size, limit int) []interface{} {}

// Get the number of queued broadcasts
func (q *BroadcastQueue) Size() int {}
```

The `BroadcastQueue` manages a priority list of pending broadcasts, with broadcasts with fewer transmissions receiving higher priority. When broadcasts have the same priority, broadcasts with smaller size are given priority to increase the number of broadcasts transmitted per matching.

The design of `BroadcastQueue` is based on the memberlist `TransmitLimitedQueue`. The transmission limit is given as an argument to `Match()` instead of as a delegate function to enforce separation of concerns. Likewise, instead of implementing a `Prune()` method to prevent unbounded queue size, when `Limit > 0`, `BroadcastQueue` instead blocks `Push()` when the queue is full. This has the effect of providing backflow control for throttling the broadcast mechanism.


## `MailHandler` for incoming messages

```go
type HandlerFunc func(message interface{}, next HandlerFunc) error

type MessageHandler struct {
    Handlers []MessageHandler
    stop     chan struct{}
}

// Start accepting packets, returning on error.
func (m *Mailman) Accept(packets <-chan IncomingPacket) error

// Deliver a single packet.
func (m *Mailman) Deliver(packet IncomingPacket) error

// Stop accepting packets.
func (m *Mailman) Stop() error
```

`MailHandler` implements the chain-of-responsibility pattern. If a `MessageHandler` is unable to handle a message, it should call the next handler.


## `FailureDetector` implements SWIM

```go
type FailureDetector struct {
    sequence    uint32
    incarnation uint32

    nodeSelectionList NodeSelectionList
    broadcastQueue    BroadcastQueue

    regionProbes   uint
    neighborProbes uint

    transport   Transport
    mailHandler MailHandler

    ticker AftershockTicker

    stop chan struct{}
}

// Send a direct message to the node.
func (d *FailureDetector) SendTo(node *Node, message interface{}) error {}

// Send a broadcast message to the group.
func (d *FailureDetector) Broadcast(message interface{}) error {}

// Use a mail handler.
func (d *FailureDetector) Use(handler HandlerFunc) error {}

// Start the failure detector.
func (d *FailureDetector) Start() error {}

// Gracefully stop the failure detector.
func (d *FailureDetector) Stop() error {}

// Immediately stop the failure detector.
func (d *FailureDetector) Close() error {}
```


## `MemberList` exposes the client API

```go
type Options struct {
    Id   Id          // Local node ID
    Meta interface{} // Initial metadata for local node

    Sorter NodeSorter // Node sorting implementation

    Transport Transport // Transport implementation

    RetransmitMult uint // Retransmits = RetransmitMult * log(N+1)
    SuspicionMult  uint // SuspicionTimeout = SuspicionMult * log(N+1) * ProbeInterval

    IndirectProbes uint // Number of indirect probes

    ProbeInterval time.Duration // Time between protocol periods
    ProbeTimeout  time.Duration // Timeout after a direct probe before using indirect probes

    RegionCount  uint // Number of regional buckets to maintain
    RegionProbes uint // Number of regional probes per protocol period

    NeighborCount  uint // Number of nodes in the neighborhood
    NeighborProbes uint // Number of neighborhood probes per protocol period
}

type MemberList struct {
    UserMessages <-chan interface{} // Received user messages are queued here
    failureDetector FailureDetector
}

// Create a new MemberList.
func New(options Options) (*MemberList, error) {}

// Broadcast a user message.
func (l *MemberList) Broadcast(data interface{}) error {}

// Get the local node.
func (l *MemberList) LocalNode() *Node {}

// Get a list of known live nodes.
func (l *MemberList) Members() []*Node {}

// Get an estimate of the number of known live nodes.
func (l *MemberList) MemberCount() int {}

// Set the local node metadata and enqueue a broadcast update.
func (l *MemberList) SetMeta(meta interface{}) error {}

// Start the failure detector.
func (l *MemberList) Start() error {}

// Gracefully stop the failure detector.
func (l *MemberList) Stop() error {}

// Broadcast the local node's state, blocking until the message has been 
// sent to a member of the group, or until the timeout expires.
//
// The failure detector must have been started before calling this method.
func (l *MemberList) Update(timeout time.Timeout) error {}

// Broadcast a join intent, blocking until the message has been sent to all
// the given addresses.
//
// The failure detector must have been started before calling this method.
func (l *MemberList) Join(addrs []string) error {}

// Broadcast a leave intent, blocking until the message has been sent to a
// member of the group, or until the timeout expires.
//
// The failure detector must have been started before calling this method.
func (l *MemberList) Leave(timeout time.Timeout) error {}
```

`MemberList` is exposes the primary user-facing API. It implements the modified SWIM failure detector described in the [README][readme]. Setting `regionCount=1`, `regionProbes=1`, `neighborCount=0`, and `neighborProbes=0` results in the original SWIM behavior. The API is based on the [memberlist][] structure of the same name.


[memberlist]: https://github.com/hashicorp/memberlist
[readme]: ../README.md
