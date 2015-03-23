package swim

import (
	"time"
)

// A ping event probes a node. The timestamp is returned in the ack event by
// the receiving node. The requesting node sends its return addresses and
// incarnation number for anti-entropy.
type PingEvent struct {
	From uint64    // ID of requesting node
	Time time.Time // Local time at ping node
}

// An ack event acknowledges a ping. The returned timestamp is used to
// determine which ping the remote node is responding to and to measure the
// round-trip time. The last known incarnation number of the requesting node
// is returned by the responding node for anti-entropy.
type AckEvent struct {
	From uint64    // ID of requesting node
	Time time.Time // Local time at ping node
}

// An indirect ping request asks an unrelated node to probe the target node.
type IndirectPingRequestEvent struct {
	From        uint64    // ID of requesting node
	Addrs       []string  // Addresses of requesting node
	Target      uint64    // ID of target node
	TargetAddrs []string  // Addresses of target node
	Time        time.Time // Local time at requesting node
}

// An indirect ping indirectly probes a node.
type IndirectPingEvent struct {
	PingEvent
	Addrs    []string  // Addresses of the pinging node
	Via      uint64    // ID of the node node requesting the ping
	ViaAddrs []string  // Addresses of node requesting the ping
	ViaTime  time.Time // Local time at node requesting the ping
}

// An indirect ping response returns the successful indirect ping for a
// target node. The timestamp is used to determine to which indirect ping
// request the remote node is responding to. The timestamp cannot be used to
// measure round-trip time due to the indirection.
type IndirectAckEvent struct {
	AckEvent
	Via     uint64    // ID of the node requesting the indirect probe
	ViaTime time.Time // Local time at requesting node
}

// An anti-entropy event updates a node to the most up-to-date incarnation
// of the target node.
type AntiEntropyEvent struct {
	From uint64
	Node
}

// Broadcast tags are used to efficiently invalidate existing broadcasts.
type BroadcastTag struct {
	Id    uint64
	State State
}

// A broadcast event exposes the sequence and tag methods.
type BroadcastEvent interface {
	Tag() BroadcastTag
	Seq() *Seq
}

// An alive event indicates that a node is alive, joining the group, or
// that its metadata (addresses and/or user data) has changed.
type AliveEvent struct {
	From uint64 // ID of the node broadcasting this event
	Node        // The alive node
}

// Get the tag for the alive event.
func (e *AliveEvent) Tag() BroadcastTag {
	return BroadcastTag{e.Id, Alive}
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
	return BroadcastTag{e.Id, Suspect}
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
	return BroadcastTag{e.Id, Dead}
}

// Get the sequence for the death event.
func (e *DeathEvent) Seq() *Seq {
	return &e.Incarnation
}

// A user event is an application-specific broadcast. The event is passed
// directly to the client application.
type UserEvent struct {
	From        uint64      // ID of the node broadcasting this event
	Incarnation Seq         // Incarnation number of the event
	Data        interface{} // User-specific data associated with the node
}

// Get the tag for the user event.
func (e *UserEvent) Tag() BroadcastTag {
	return BroadcastTag{uint64(e.Incarnation), 0}
}

// Get the sequence for the user event.
func (e *UserEvent) Seq() *Seq {
	return &e.Incarnation
}
