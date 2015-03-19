package swim

// A broadcast describes an event to be broadcast to the group.
type Broadcast struct {
	Class    int           // Priority class of the broadcast
	Attempts int           // Number of transmissions attempted
	Event    interface{}   // The event to broadcast
	Done     chan struct{} // The channel on which to signal done
}

// Calculate the overall priority for the broadcast. Broadcasts with lower
// class values have priority over higher class values. For example, a
// broadcast with class 1 has higher priority than a broadcast with class 2.
// The overall priority is calculated as:
//
//   Priority = Class * Attempts
//
// Thus, a broadcast with class 0 always has higher priority, regardless of
// the number of transmission attempts.
func (b *Broadcast) Priority() int {
	return b.Class * b.Attempts
}

// Determine if this broadcast invalidates that broadcast.
func (b *Broadcast) Invalidates(that *Broadcast) bool {
	return b.tag() == that.tag() && that.seq().Compare(b.seq().Get()) < 0
}

// Get the sequence number for the event.
func (b *Broadcast) seq() *Seq {
	switch e := b.Event.(type) {
	case *AliveEvent:
		return &e.Incarnation

	case *SuspectEvent:
		return &e.Incarnation

	case *DeathEvent:
		return &e.Incarnation

	case *UserEvent:
		return &e.Seq

	default:
		panic("unknown broadcast type")
	}
}

const (
	_ = iota
	bcastAlive
	bcastSuspect
	bcastDeath
	bcastUser
)

// Broadcast tags are used to efficiently invalidate existing broadcasts.
type bcastTag struct {
	From uint64
	Id   uint64
	Type int
}

// Create a tag for the broadcast.
func (b *Broadcast) tag() (tag bcastTag) {

	switch e := b.Event.(type) {
	case *AliveEvent:
		tag.From = e.From
		tag.Id = e.Node.Id
		tag.Type = bcastAlive

	case *SuspectEvent:
		tag.From = e.From
		tag.Id = e.Id
		tag.Type = bcastSuspect

	case *DeathEvent:
		tag.From = e.From
		tag.Id = e.Id
		tag.Type = bcastDeath

	case *UserEvent:
		tag.From = e.From
		tag.Id = uint64(e.Seq)
		tag.Type = bcastUser

	default:
		panic("unknown broadcast type")
	}

	return
}
