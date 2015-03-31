package swim

import (
	"sort"
)

// A broadcast queue implements a priority queue ordered on the number of
// transmission attempts and the priority class of the broadcasts.
type BroadcastQueue struct {
	live   map[BroadcastTag]*Broadcast // Broadcasts by tag for invalidation
	sorted byPriority                  // Broadcasts queue in list form
	order  int
}

func NewBroadcastQueue() *BroadcastQueue {
	return &BroadcastQueue{
		live: make(map[BroadcastTag]*Broadcast),
	}
}

// Push a broadcast onto the queue.
func (q *BroadcastQueue) Push(bcast *Broadcast) {

	// for stable sort
	bcast.order = q.order
	q.order += 1

	// invalidate an existing broadcast or add it to the queue
	tag := bcast.Event.Tag()
	if that, ok := q.live[tag]; ok {
		if bcast.Invalidates(that) {

			// signal we're done
			if that.Done != nil {
				that.Done <- struct{}{}
			}

			// replace
			q.live[tag] = bcast
		}
	} else {
		q.live[tag] = bcast
	}

	// not sorted
	q.sorted = nil
}

// Remove the broadcasts for which the predicate returns true.
func (q *BroadcastQueue) Prune(predicate func(b *Broadcast) bool) {
	for tag, bcast := range q.live {
		if predicate(bcast) {
			delete(q.live, tag)
			// signal we're done
			if bcast.Done != nil {
				bcast.Done <- struct{}{}
			}
		}
	}
	q.sorted = nil
}

// Get the queue as a list ordered by priority. If any broadcasts in the
// list are modified, the caller is responsible for calling q.Init().
func (q *BroadcastQueue) List() []*Broadcast {
	if q.sorted == nil {
		for _, bcast := range q.live {
			q.sorted = append(q.sorted, bcast)
		}
		sort.Sort(q.sorted)
	}
	return q.sorted
}

// Get the number of queued broadcasts.
func (q *BroadcastQueue) Len() int {
	return len(q.live)
}

// Private type for sorting.
type byPriority []*Broadcast

// Get the number of queued broadcasts.
func (q byPriority) Len() int {
	return len(q)
}

// Determine if the broadcast at index i has a higher priority (smaller
// value) than the broadcast at index j.
func (q byPriority) Less(i, j int) bool {
	if a, b := q[i].Priority(), q[j].Priority(); a == b {
		return q[i].order < q[j].order
	} else {
		return a < b
	}
}

// Swap the broadcast at index i with the broadcast at index j.
func (q byPriority) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}
