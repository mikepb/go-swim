package swim

const mbBufSizes = 8

// Broker handles message transport and encoding.
type Broker struct {
	Codec          Codec      // The codec implementation to use
	Transport      Transport  // The transport implementation to use
	BroadcastLimit int        // The broadcast transmission limit
	Errors         chan error // Channel on which to send errors

	Inbox <-chan *Message // Queue of incoming messages
	inbox chan *Message

	bqueue    BroadcastQueue // Broadcast queue
	bEstimate float64        // Estimate of the number of broadcasts to send

	state chan bool
}

// Create a new broker using the given codec and transport.
func NewBroker(c Codec, t Transport) *Broker {
	inbox := make(chan *Message, mbBufSizes)
	b := &Broker{
		Codec:     c,
		Transport: t,

		Inbox: inbox,
		inbox: inbox,

		state: make(chan bool, 1),
	}

	// empty = running
	// true  = stop signal
	// false = stopped
	b.state <- false

	return b
}

// Start delivering messages to individual channels.
func (b *Broker) Start() {

	// don't call multiple times!
	if len(b.state) == 0 || <-b.state {
		panic("already started")
	}

	// start handling messages asynchronously
	go func() {
		for {
			select {
			case coded := <-b.Transport.Inbox():
				b.deliver(coded)
			case <-b.state:
				b.state <- false
				return
			}
		}
	}()
}

func (b *Broker) deliver(coded *CodedMessage) {

	// decode message
	if err := b.Codec.Decode(coded); err != nil {
		b.deliverError(err)
		return
	}

	// deliver message
	b.inbox <- coded.Message
}

// Stop handling messages.
func (b *Broker) Stop() {

	// don't call multiple times!
	if len(b.state) != 0 {
		panic("already stopped")
	}

	// signal goroutine to stop
	b.state <- true

	// receive acknowledgement
	b.state <- <-b.state
}

// Send a direct message to the given node without piggybacking broadcasts.
func (b *Broker) DirectTo(node *Node, msg *Message) {
	coded := &CodedMessage{Message: msg}

	// encode the message without piggybacked broadcasts
	if err := b.Codec.Encode(coded); err != nil {
		b.deliverError(err)
		return
	}

	// send the message
	b.sendTo(node, coded)
}

// Send a message to the given node. Broadcasts are piggybacked to the
// message up to the message size limit.
func (b *Broker) SendTo(node *Node, msg *Message) {
	coded := &CodedMessage{Message: msg}

	// encode the message with piggybacked broadcasts
	if err := b.encodeWithBroadcasts(coded); err != nil {
		b.deliverError(err)
		return
	}

	// send the message
	b.sendTo(node, coded)
}

// Encode the given message after piggybacking broadcasts.
func (b *Broker) encodeWithBroadcasts(coded *CodedMessage) error {

	// attach broadcasts
	if bcasts := b.bqueue.List(); len(bcasts) > 0 {

		// limit number of piggybacked broadcasts if supported
		if b.Transport.MaxMessageLen() > 0 && b.bEstimate > 0.0 {
			end := int(b.bEstimate) - coded.Message.EventCount()

			// attach at least one event
			if end < 1 {
				end = 1
			}

			// slice it
			if end < len(bcasts) {
				bcasts = bcasts[:end]
			}
		}

		// add the events
		for _, bcast := range bcasts {
			coded.Message.AddEvent(bcast)
			bcast.Attempts += 1
		}

		// prune the queue and re-sort
		b.bqueue.Prune(func(bcast *Broadcast) bool {
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
		count := float64(coded.Message.EventCount())
		b.bEstimate = 0.75*b.bEstimate + 0.25*(maxSize/(size/count))

	} else {

		// encoder does not support size
		b.bEstimate = -1.0

	}

	return nil
}

// Send the message to the given node.
func (b *Broker) sendTo(node *Node, coded *CodedMessage) {
	if err := b.Transport.SendTo(node, coded); err != nil {
		b.deliverError(err)
	}
}

// Notify client of an error.
func (b *Broker) deliverError(err error) {
	if b.Errors != nil {
		b.Errors <- err
	}
}
