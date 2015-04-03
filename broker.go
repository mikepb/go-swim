package swim

import (
	"sync"
	"sync/atomic"
)

// Broker handles piggybacked broadcast messages, transport, and encoding.
// The methods on this object are safe to call from multiple goroutines.
type Broker struct {
	Transport                  // The transport implementation to use
	Codec      Codec           // The codec implementation to use
	Broadcasts *BroadcastQueue // Broadcast queue
	l          sync.Mutex      // Broadcast lock.
	bEstimate  float64         // Estimate of the number of broadcasts to send
	limit      uint32          // The broadcast transmission limit
}

// Create a new broker.
func NewBroker(transport Transport, codec Codec) *Broker {
	return &Broker{
		Transport:  transport,
		Codec:      codec,
		Broadcasts: NewBroadcastQueue(),
		bEstimate:  2.0,
	}
}

func (b *Broker) SetBroadcastLimit(limit uint) {
	atomic.StoreUint32(&b.limit, uint32(limit))
}

func (b *Broker) BroadcastLimit() uint {
	return uint(atomic.LoadUint32(&b.limit))
}

// Receive and decode a message from the network.
func (b *Broker) Recv() (*Message, error) {

	// receive message
	coded, err := b.Transport.Recv()
	if err != nil {
		return nil, err
	}

	// decode message
	if err := b.Codec.Decode(coded); err != nil {
		return nil, err
	}

	// deliver message
	return &coded.Message, nil
}

// Send a direct message to the node represented by the given address
// without piggybacking broadcasts.
func (b *Broker) DirectTo(addrs []string, msg *Message) error {
	coded := &CodedMessage{Message: *msg}

	// encode the message without piggybacked broadcasts
	if err := b.Codec.Encode(coded); err != nil {
		return err
	}

	// send the message
	return b.Transport.SendTo(addrs, coded)
}

// Send a message to the node represented by the given addresses. Broadcasts
// are piggybacked to the message up to the message size limit.
func (b *Broker) SendTo(addrs []string, msg *Message) error {
	coded := &CodedMessage{Message: *msg}

	// encode the message with piggybacked broadcasts
	if err := b.encodeWithBroadcasts(coded); err != nil {
		return err
	}

	// send the message
	return b.Transport.SendTo(addrs, coded)
}

// Encode the given message after piggybacking broadcasts.
func (b *Broker) encodeWithBroadcasts(coded *CodedMessage) error {

	// lock for concurrent access
	b.l.Lock()
	defer b.l.Unlock()

	// attach broadcasts
	if bcasts := b.Broadcasts.List(); len(bcasts) > 0 {
		max := len(bcasts)

		// limit number of piggybacked broadcasts if supported
		if b.Transport.MaxMessageLen() > 0 && b.bEstimate > 0.0 {
			if i := int(b.bEstimate) - len(coded.Message.Events()); i < 1 {
				// attach at least one event
				max = 1
			} else if i < max {
				// bound to number of broadcasts
				max = i
			}
		}

		// add the events
		for _, bcast := range bcasts[:max] {
			// lazy initialize state
			if bcast.State == nil {
				bcast.State = make(map[uint64]struct{})
				bcast.State[bcast.Event.Source()] = struct{}{}
			}
			// don't send more broadcast to source or the same node
			if _, ok := bcast.State[coded.Message.To]; !ok {
				coded.Message.AddEvent(bcast.Event)
				bcast.Attempts += 1
				bcast.State[coded.Message.To] = struct{}{}
			}
		}

		// prune the queue and re-sort
		limit := b.BroadcastLimit()
		b.Broadcasts.Prune(func(bcast *Broadcast) bool {
			return bcast.Attempts >= limit
		})
	}

	// encode the message
	if err := b.Codec.Encode(coded); err != nil {
		return err
	}

	// update estimate of the number of messages to encode
	if coded.Size > 0 {

		// estimate number of events supported
		maxSize := float64(b.Transport.MaxMessageLen())
		size := float64(coded.Size)
		count := float64(len(coded.Message.Events()))
		b.bEstimate = 0.75*b.bEstimate + 0.25*(maxSize/(size/count))

		// bias the estimate towards including more broadcasts
		if size < maxSize {
			b.bEstimate += 0.1
		}

	} else {

		// encoder does not support size
		b.bEstimate = -1.0

	}

	return nil
}

// Queue a broadcast event.
func (b *Broker) Broadcast(event BroadcastEvent) {
	b.broadcastWithPriority(event, 2)
}

// Queue a low-priority broadcast event.
func (b *Broker) BroadcastL(event BroadcastEvent) {
	b.broadcastWithPriority(event, 3)
}

func (b *Broker) broadcastWithPriority(event BroadcastEvent, prio uint) {

	// lock for concurrent access
	b.l.Lock()
	defer b.l.Unlock()

	// add broadcast to queue
	b.Broadcasts.Push(&Broadcast{Class: prio, Event: event})
}

// Broadcast an event and notify on the done channel when the broadcast is
// removed from the queue, either from invalidation or after reaching the
// broadcast transmission limit.
func (b *Broker) BroadcastSync(event BroadcastEvent) chan struct{} {
	done := make(chan struct{}, 1)

	// lock for concurrent access
	b.l.Lock()
	defer b.l.Unlock()

	// add broadcast to queue with high priority
	b.Broadcasts.Push(&Broadcast{Class: 1, Event: event, Done: done})

	// return async channel
	return done
}
