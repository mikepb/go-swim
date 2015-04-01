package swim

import (
	"fmt"
	"time"
)

// A ping event probes a node. The timestamp is returned in the ack event by
// the receiving node. The requesting node sends its return addresses and
// incarnation number for anti-entropy.
type PingEvent struct {
	From uint64    // ID of requesting node
	Time time.Time // Local time at ping node
}

// Default format output.
func (e PingEvent) String() string {
	return fmt.Sprintf("PingEvent{ From: %v, Time: %v }", e.From, e.Time)
}

// An ack event acknowledges a ping. The returned timestamp is used to
// determine which ping the remote node is responding to and to measure the
// round-trip time. The last known incarnation number of the requesting node
// is returned by the responding node for anti-entropy.
type AckEvent struct {
	From uint64    // ID of requesting node
	Time time.Time // Local time at ping node
}

// Default format output.
func (e AckEvent) String() string {
	return fmt.Sprintf("AckEvent{ From: %v, Time: %v }", e.From, e.Time)
}

// An indirect ping request asks an unrelated node to probe the target node.
type IndirectPingRequestEvent struct {
	From        uint64    // ID of requesting node
	Addrs       []string  // Addresses of requesting node
	Target      uint64    // ID of target node
	TargetAddrs []string  // Addresses of target node
	Time        time.Time // Local time at requesting node
}

// Default format output.
func (e IndirectPingRequestEvent) String() string {
	return fmt.Sprintf(
		"IndirectPingRequestEvent{ From: %v, Addrs: %v, Target: %v, TargetAddrs: %v, Time: %v }",
		e.From, e.Addrs, e.Target, e.TargetAddrs, e.Time)
}

// An indirect ping indirectly probes a node.
type IndirectPingEvent struct {
	PingEvent
	Addrs    []string  // Addresses of the pinging node
	Via      uint64    // ID of the node node requesting the ping
	ViaAddrs []string  // Addresses of node requesting the ping
	ViaTime  time.Time // Local time at node requesting the ping
}

// Default format output.
func (e IndirectPingEvent) String() string {
	return fmt.Sprintf(
		"IndirectPingEvent{ %v, Addrs: %v, Via: %v, ViaAddrs: %v, ViaTime: %v }",
		e.PingEvent, e.Addrs, e.Via, e.ViaAddrs, e.ViaTime)
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

// Default format output.
func (e IndirectAckEvent) String() string {
	return fmt.Sprintf(
		"IndirectAckEvent{ %v, Via: %v, ViaTime: %v }",
		e.AckEvent, e.Via, e.ViaTime)
}

// An anti-entropy event updates a node to the most up-to-date incarnation
// of the target node. The sender is always the node described in the event.
type AntiEntropyEvent struct {
	Node
}

// Default format output.
func (e AntiEntropyEvent) String() string {
	return fmt.Sprintf("AntiEntropyEvent{ %v }", e.Node)
}

// Broadcast tags are used to efficiently invalidate existing broadcasts.
type BroadcastTag struct {
	Id      uint64
	IsState bool
}

// A broadcast event exposes the sequence and tag methods.
type BroadcastEvent interface {
	Source() uint64
	Tag() BroadcastTag
	Seq() *Seq
}

// An alive event indicates that a node is alive, joining the group, or
// that its metadata (addresses and/or user data) has changed.
type AliveEvent struct {
	From uint64 // ID of the node broadcasting this event
	Node        // The alive node
}

// Default format output.
func (e AliveEvent) String() string {
	return fmt.Sprintf("AliveEvent{ From: %v, %v }", e.From, e.Node)
}

// Get the source for this broadcast event.
func (e AliveEvent) Source() uint64 {
	return e.From
}

// Get the tag for the alive event.
func (e AliveEvent) Tag() BroadcastTag {
	return BroadcastTag{e.Id, true}
}

// Get the sequence for the alive event.
func (e AliveEvent) Seq() *Seq {
	return &e.Incarnation
}

// A suspect event indicates that a node is suspected of death.
type SuspectEvent struct {
	From        uint64 // ID of the node broadcasting this event
	Id          uint64 // ID of the suspected node
	Incarnation Seq    // Incarnation number of the node
}

// Default format output.
func (e SuspectEvent) String() string {
	return fmt.Sprintf(
		"SuspectEvent{ From: %v, Id: %v, Incarnation: %v }",
		e.From, e.Id, e.Incarnation)
}

// Get the source for this broadcast event.
func (e SuspectEvent) Source() uint64 {
	return e.From
}

// Get the tag for the suspect event.
func (e SuspectEvent) Tag() BroadcastTag {
	return BroadcastTag{e.Id, true}
}

// Get the sequence for the suspect event.
func (e SuspectEvent) Seq() *Seq {
	return &e.Incarnation
}

// A death event indicates that a node is confirmed dead.
type DeathEvent struct {
	From        uint64 // ID of the node broadcasting this event
	Id          uint64 // ID of the dead node
	Incarnation Seq    // Incarnation number of the node
}

// Default format output.
func (e DeathEvent) String() string {
	return fmt.Sprintf(
		"DeathEvent{ From: %v, Id: %v, Incarnation: %v }",
		e.From, e.Id, e.Incarnation)
}

// Get the source for this broadcast event.
func (e DeathEvent) Source() uint64 {
	return e.From
}

// Get the tag for the death event.
func (e DeathEvent) Tag() BroadcastTag {
	return BroadcastTag{e.Id, true}
}

// Get the sequence for the death event.
func (e DeathEvent) Seq() *Seq {
	return &e.Incarnation
}

// A user event is an application-specific broadcast. The event is passed
// directly to the client application.
type UserEvent struct {
	From        uint64      // ID of the node broadcasting this event
	Incarnation Seq         // Incarnation number of the event
	Data        interface{} // User-specific data associated with the node
}

// Default format output.
func (e UserEvent) String() string {
	return fmt.Sprintf(
		"UserEvent{ From: %v, Incarnation: %v, %v }",
		e.From, e.Incarnation, e.Data)
}

// Get the source for this broadcast event.
func (e UserEvent) Source() uint64 {
	return e.From
}

// Get the tag for the user event.
func (e UserEvent) Tag() BroadcastTag {
	return BroadcastTag{uint64(e.Incarnation), false}
}

// Get the sequence for the user event.
func (e UserEvent) Seq() *Seq {
	return &e.Incarnation
}
