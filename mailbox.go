package swim

const mbBufSizes = 8

// Mailbox handles message transport and encoding.
type Mailbox struct {
	Codec     Codec           // The codec implementation to use
	Transport Transport       // The transport implementation to use
	Inbox     <-chan *Message // Queue of incoming messages.
	Errors    <-chan error    // Last error encountered.

	inbox  chan *Message
	errors chan error

	running  bool
	stopping chan struct{}
	stopped  chan struct{}
}

// Create a new mailbox using the given codec and transport.
func NewMailbox(c Codec, t Transport) *Mailbox {
	inbox := make(chan *Message, mbBufSizes)
	errs := make(chan error, 1)

	return &Mailbox{
		Codec:     c,
		Transport: t,

		Inbox: inbox,
		inbox: inbox,

		Errors: errs,
		errors: errs,

		stopping: make(chan struct{}, 1),
		stopped:  make(chan struct{}, 1),
	}
}

// Start delivering messages to individual channels.
func (m *Mailbox) Start() {

	// don't call multiple times!
	if m.running {
		panic("already started")
	}

	// hint to ourselves that we've started; user should not call from
	// multiple goroutines
	m.running = true

	// start handling messages asynchronously
	go func() {
		for {
			select {
			case coded := <-m.Transport.Inbox():
				m.deliver(coded)
			case <-m.stopping:
				m.stopped <- struct{}{}
				return
			}
		}
	}()
}

func (m *Mailbox) deliver(coded *CodedMessage) {

	// decode message
	if err := m.Codec.Decode(coded); err != nil {
		m.deliverError(err)
		return
	}

	// deliver message
	m.inbox <- coded.Message
}

// Stop handling messages.
func (m *Mailbox) Stop() {

	// don't call multiple times!
	if !m.running {
		panic("already stopped")
	}

	// signal goroutine to stop
	m.stopping <- struct{}{}

	// receive acknowledgement
	<-m.stopped

	// hint to ourselves that we've stopped; user should not call from
	// multiple goroutines
	m.running = false
}

// Send a message to the given node.
func (m *Mailbox) SendTo(node *Node, msg *Message) {

	// encode the message
	coded := &CodedMessage{Message: msg}
	if err := m.Codec.Encode(coded); err != nil {
		m.deliverError(err)
		return
	}

	// send the message
	if err := m.Transport.SendTo(node, coded); err != nil {
		m.deliverError(err)
	}
}

func (m *Mailbox) deliverError(err error) {
	if len(m.errors) != 0 {
		<-m.errors
	}
	m.errors <- err
}

// Get a hit of the maximum message size supported by the underlying
// transport.
func (m *Mailbox) MaxMessageSize() int {
	return m.Transport.MaxMessageSize()
}
