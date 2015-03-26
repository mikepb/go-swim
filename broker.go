package swim

// Broker handles piggybacked broadcast messages, transport, and encoding.
type Broker struct {
	Transport                      // The transport implementation to use
	Codec          Codec           // The codec implementation to use
	BroadcastLimit uint            // The broadcast transmission limit
	Broadcasts     *BroadcastQueue // Broadcast queue
	bEstimate      float64         // Estimate of the number of broadcasts to send
}

// Create a new broker.
func NewBroker(transport Transport, codec Codec) *Broker {
	return &Broker{
		Transport:  transport,
		Codec:      codec,
		Broadcasts: NewBroadcastQueue(),
	}
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

	// attach broadcasts
	if bcasts := b.Broadcasts.List(); len(bcasts) > 0 {
		max := len(bcasts)

		// limit number of piggybacked broadcasts if supported
		if b.Transport.MaxMessageLen() > 0 && b.bEstimate > 0.0 {
			i := int(b.bEstimate) - len(coded.Message.Events())

			// attach at least one event
			if i < 1 {
				i = 1
			}

			// update max if less
			if i < max {
				max = i
			}
		}

		// add the events
		for _, bcast := range bcasts[:max] {
			// don't send broadcast to source
			if coded.Message.To != bcast.Event.Source() {
				coded.Message.AddEvent(bcast.Event)
				bcast.Attempts += 1
			}
		}

		// prune the queue and re-sort
		b.Broadcasts.Prune(func(bcast *Broadcast) bool {
			return bcast.Attempts >= b.BroadcastLimit
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

	} else {

		// encoder does not support size
		b.bEstimate = -1.0

	}

	return nil
}

// Queue a broadcast event.
func (b *Broker) Broadcast(event BroadcastEvent) {
	b.Broadcasts.Push(&Broadcast{Class: 2, Event: event})
}

// Broadcast an event and notify on the done channel when the broadcast is
// removed from the queue, either from invalidation or after reaching the
// broadcast transmission limit.
func (b *Broker) BroadcastSync(event BroadcastEvent) chan struct{} {
	done := make(chan struct{}, 1)
	b.Broadcasts.Push(&Broadcast{Class: 1, Event: event, Done: done})
	return done
}
