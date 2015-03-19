package swim

// A broadcast describes an event to be broadcast to the group.
type Broadcast struct {
	Class    int            // Priority class of the broadcast
	Attempts int            // Number of transmissions attempted
	Event    BroadcastEvent // The event to broadcast
	Done     chan struct{}  // The channel on which to signal done
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
	return b.Event.Tag() == that.Event.Tag() &&
		that.Event.Seq().Compare(b.Event.Seq().Get()) < 0
}
