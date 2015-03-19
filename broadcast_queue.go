package swim

import (
	"sort"
)

// A broadcast queue implements a priority queue ordered on the number of
// transmission attempts and the priority class of the broadcasts.
type BroadcastQueue struct {
	sourceMap map[BroadcastTag]*Broadcast // Broadcasts by tag for invalidation
	bcasts    bcastQueue                  // Broadcasts queue in list form
	sorted    bool                        // Hint that the queue is sorted
}

func NewBroadcastQueue() *BroadcastQueue {
	return &BroadcastQueue{
		sourceMap: make(map[BroadcastTag]*Broadcast),
	}
}

// Initialize the priority queue by fixing the sort order.
func (q *BroadcastQueue) Init() {
	sort.Stable(q.bcasts)

	// prune removed broadcasts
	end := len(q.bcasts)
	for end > 0 && q.bcasts[end-1] == nil {
		end -= 1
	}
	q.bcasts = q.bcasts[:end]

	// hint that queue is sorted
	q.sorted = true
}

// Maybe sort the queue if the sorted hint is false.
func (q *BroadcastQueue) maybeSort() {
	if !q.sorted {
		q.Init()
		q.sorted = true
	}
}

// Return the highest priority broadcast without removing it.
func (q *BroadcastQueue) Peek() *Broadcast {
	if len(q.bcasts) == 0 {
		return nil
	}
	q.maybeSort()
	return q.bcasts[0]
}

// Push a broadcast onto the queue.
func (q *BroadcastQueue) Push(bcast *Broadcast) {

	// invalidate an existing broadcast or add it to the queue
	tag := bcast.Event.Tag()
	if that, ok := q.sourceMap[tag]; ok {
		if bcast.Invalidates(that) {
			*that = *bcast
		}
	} else {
		q.sourceMap[tag] = bcast
		q.bcasts = append(q.bcasts, bcast)
	}

	// hint that queue is unsorted
	q.sorted = false
}

// Remove the highest priority broadcast from the queue.
func (q *BroadcastQueue) Pop() *Broadcast {
	q.maybeSort()
	bcast := q.bcasts[0]
	q.bcasts = q.bcasts[1:]
	delete(q.sourceMap, bcast.Event.Tag())
	return bcast
}

// Remove the broadcasts for which the predicate returns true.
func (q *BroadcastQueue) Prune(predicate func(b *Broadcast) bool) {
	for i, b := range q.bcasts {
		if predicate(b) {
			q.bcasts[i] = nil
		}
	}
	q.Init()
}

// Get the queue as a list ordered by priority. If any broadcasts in the
// list are modified, the caller is responsible for calling q.Init().
func (q *BroadcastQueue) List() []*Broadcast {
	q.maybeSort()
	return q.bcasts
}

// Get the number of queued broadcasts.
func (q *BroadcastQueue) Len() int {
	return len(q.bcasts)
}

// Private type for sorting.
type bcastQueue []*Broadcast

// Get the number of queued broadcasts.
func (q bcastQueue) Len() int {
	return len(q)
}

// Determine if the broadcast at index i has a higher priority (smaller
// value) than the broadcast at index j.
func (q bcastQueue) Less(i, j int) bool {

	// handle nil for pruning
	if q[i] == nil {
		if q[j] == nil {
			return i < j
		}
		return false
	}
	if q[j] == nil {
		return true
	}

	return q[i].Priority() < q[j].Priority()
}

// Swap the broadcast at index i with the broadcast at index j.
func (q bcastQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}
