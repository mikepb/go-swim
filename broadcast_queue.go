package swim

import (
	"container/heap"
)

// A broadcast queue implements a priority queue ordered on the number of
// transmission attempts and the priority class of the broadcasts.
type BroadcastQueue struct {
	sourceMap map[bcastTag]*Broadcast
	bcasts    bcastQueue
}

func NewBroadcastQueue() *BroadcastQueue {
	return &BroadcastQueue{
		sourceMap: make(map[bcastTag]*Broadcast),
	}
}

// Initialize the priority queue by fixing the heap invariants.
func (q *BroadcastQueue) Init() {
	heap.Init(&q.bcasts)
}

// Fix the heap invariants resulting from a change to the given broadcast.
func (q *BroadcastQueue) Fix(bcast *Broadcast) {
	q.Init()
}

// Return the highest priority broadcast without removing it.
func (q *BroadcastQueue) Peek() *Broadcast {
	if len(q.bcasts) == 0 {
		return nil
	}
	return q.bcasts[0]
}

// Push a broadcast onto the queue.
func (q *BroadcastQueue) Push(bcast *Broadcast) {

	// broadcast does not belong to a queue or it belongs to this one
	if bcast.q != nil && bcast.q != q {
		panic("broadcast belongs to a different queue")
	}

	// this broadcast now belongs to this queue
	bcast.q = q

	// invalidate an existing broadcast or add it to the queue
	tag := bcast.tag()
	if that, ok := q.sourceMap[tag]; ok {
		if bcast.Invalidates(that) {
			*that = *bcast
			q.Init()
		}
	} else {
		q.sourceMap[tag] = bcast
		q.bcasts = append(q.bcasts, bcast)
	}
}

// Remove the highest priority broadcast from the queue.
func (q *BroadcastQueue) Pop() *Broadcast {
	bcast := heap.Pop(&q.bcasts).(*Broadcast)
	bcast.q = nil
	delete(q.sourceMap, bcast.tag())
	return bcast
}

// Get the number of queued broadcasts.
func (q *BroadcastQueue) Len() int {
	return len(q.bcasts)
}

type bcastQueue []*Broadcast

// Get the number of queued broadcasts.
func (q bcastQueue) Len() int {
	return len(q)
}

// Determine if the broadcast at index i has a lower priority than the
// broadcast at index j.
func (q bcastQueue) Less(i, j int) bool {
	return q[i].Priority() < q[j].Priority()
}

// Swap the broadcast at index i with the broadcast at index j.
func (q bcastQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

// Implementation of heap.Interface.Pop().
func (q *bcastQueue) Push(x interface{}) {
	bcast := x.(*Broadcast)
	*q = append(*q, bcast)
}

// Implementation of heap.Interface.Pop().
func (q *bcastQueue) Pop() interface{} {
	old := *q
	l := len(old) - 1
	b := old[l]
	*q = old[:l]
	return b
}
