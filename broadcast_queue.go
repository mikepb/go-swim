package swim

type Broadcast interface {

	// Determine if another broadcast invalidates this one.
	Invalidates(b Broadcast) bool

	// Get the broadcast data.
	Data() interface{}

	// Get an estimate of the broadcast size.
	Size() int

	// Invoked when the broadcast reaches its retransmission limit or is
	// invalidated by another broadcast.
	Done()
}

type limitedBroadcast struct {
	transmits int       // Number of transmissions attempted
	b         Broadcast // The broadcast
}

// Implements the sort.Interface methods
type priorityBroadcasts []*limitedBroadcast

type BroadcastQueue struct {
	Limit     int // Maximum number of queued broadcasts before blocking
	Transmits int // Maximum number of transmission
	queue     priorityBroadcasts
}

// Set the transmission limit, pruning the queue of broadcasts exceeding
// this limit.
func (q *BroadcastQueue) SetTransmits(transmits int) {
	q.Transmits = transmits
}

// Push a broadcast onto the queue, blocking if Limit > 0 and the queue is
// full. Broadcasts exceeding the transmission limit are removed from the
// queue before adding the new broadcast.
func (q *BroadcastQueue) Push(bs ...Broadcast) {}

// Consume the broadcasts, incrementing the number of transmissions and
// removing the broadcast from the queue if it exceeds the transmission limit.
func (q *BroadcastQueue) Consume(bs ...Broadcast) {}

// Match broadcasts up to the given byte size and broadcast transmission
// limit, if set. A byte size of zero or less means no size limit.
func (q *BroadcastQueue) Match(size int) []Broadcast {}

// Get the number of queued broadcasts.
func (q *BroadcastQueue) Size() int {}
