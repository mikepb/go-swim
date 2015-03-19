package swim

// A ping event probes a node. The timestamp is returned in the ack event by
// the receiving node.
type PingEvent struct {
	Timestamp int64 // Local UNIX timestamp in nanoseconds at ping node
	// TODO: if Byzantine nodes are present, a signed timestamp will provide
	// partial protection
}

// An ack event acknowledges a ping. The returned timestamp is used to
// determine which ping the remote node is responding to and to measure the
// round-trip time.
type AckEvent struct {
	Timestamp int64 // Local UNIX timestamp in nanoseconds at ping node
}

// An indirect ping request asks an unrelated node to probe the target node.
type RequestEvent struct {
	Id        uint64   // ID of target node
	Addrs     []string // Addresses for target node
	Timestamp int64    // Local UNIX timestamp in nanoseconds at ping node
}

// An indirect ping response returns the successful indirect ping for a
// target node. The timestamp is used to determine to which indirect ping
// request the remote node is responding to. The timestamp cannot be used to
// measure round-trip time due to the indirection.
type ResponseEvent struct {
	Id        uint64 // ID of target node
	Timestamp int64  // Local UNIX timestamp in nanoseconds at ping node
}

const (
	_ = iota
	bcastAlive
	bcastSuspect
	bcastDeath
	bcastUser
)

// Broadcast tags are used to efficiently invalidate existing broadcasts.
type BroadcastTag struct {
	From uint64
	Id   uint64
	Type int
}

// A broadcast event exposes the sequence and tag methods.
type BroadcastEvent interface {
	Tag() BroadcastTag
	Seq() *Seq
}

// An alive event indicates that a node is alive, joining the group, or
// that its metadata (addresses and/or user data) has changed.
type AliveEvent struct {
	From        uint64 // ID of the node broadcasting this event
	Node               // The alive node
	Incarnation Seq    // Incarnation number of the node
}

// Get the tag for the alive event.
func (e *AliveEvent) Tag() BroadcastTag {
	return BroadcastTag{e.From, e.Id, bcastAlive}
}

// Get the sequence for the alive event.
func (e *AliveEvent) Seq() *Seq {
	return &e.Incarnation
}

// A suspect event indicates that a node is suspected of death.
type SuspectEvent struct {
	From        uint64 // ID of the node broadcasting this event
	Id          uint64 // ID of the suspected node
	Incarnation Seq    // Incarnation number of the node
}

// Get the tag for the suspect event.
func (e *SuspectEvent) Tag() BroadcastTag {
	return BroadcastTag{e.From, e.Id, bcastSuspect}
}

// Get the sequence for the suspect event.
func (e *SuspectEvent) Seq() *Seq {
	return &e.Incarnation
}

// A death event indicates that a node is confirmed dead.
type DeathEvent struct {
	From        uint64 // ID of the node broadcasting this event
	Id          uint64 // ID of the dead node
	Incarnation Seq    // Incarnation number of the node
}

// Get the tag for the death event.
func (e *DeathEvent) Tag() BroadcastTag {
	return BroadcastTag{e.From, e.Id, bcastDeath}
}

// Get the sequence for the death event.
func (e *DeathEvent) Seq() *Seq {
	return &e.Incarnation
}

// A user event is an application-specific broadcast. The event is passed
// directly to the client application.
type UserEvent struct {
	From     uint64      // ID of the node broadcasting this event
	Sequence Seq         // Sequence number of the event
	Data     interface{} // User-specific data associated with the node
}

// Get the tag for the user event.
func (e *UserEvent) Tag() BroadcastTag {
	return BroadcastTag{e.From, uint64(e.Sequence), bcastUser}
}

// Get the sequence for the user event.
func (e *UserEvent) Seq() *Seq {
	return &e.Sequence
}
