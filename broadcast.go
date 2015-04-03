package swim

// A broadcast describes an event to be broadcast to the group.
type Broadcast struct {
	Class    uint                // Priority class of the broadcast
	Attempts uint                // Number of transmissions attempted
	Event    BroadcastEvent      // The event to broadcast
	Done     chan struct{}       // The channel on which to signal done
	State    map[uint64]struct{} // For avoiding sending to the same node
	order    int
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
func (b *Broadcast) Priority() uint {
	return b.Class * b.Attempts
}

// Determine if this broadcast invalidates that broadcast.
func (b *Broadcast) Invalidates(that *Broadcast) bool {
	ltag := b.Event.Tag()
	rtag := that.Event.Tag()
	if ltag != rtag {
		return false
	}

	if cmp := that.Event.Seq().Compare(b.Event.Seq().Get()); cmp < 0 {
		return true
	} else if cmp == 0 {
		// handle invalidation based on event
		switch b.Event.(type) {
		case DeathEvent:
			return true
		case SuspectEvent:
			_, ok := that.Event.(DeathEvent)
			return !ok
		}
	}
	return false
}
