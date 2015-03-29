package swim

import (
	"errors"
)

const kRecvChSize = 8

// SimTransport implements a Transport suitable for use with the simulator.
type SimTransport struct {
	Router *SimRouter
	RecvCh chan *CodedMessage
	Closed bool
}

// Create a new SimTransport associated with the given SimRouter.
func NewSimTransport(h *SimRouter) *SimTransport {
	return &SimTransport{
		Router: h,
		RecvCh: make(chan *CodedMessage, kRecvChSize),
	}
}

// Return the maximum message length hint configured in the SimRouter.
func (t *SimTransport) MaxMessageLen() int {
	return t.Router.MaxMessageLen
}

// Send a message to the transports described by the addresses using the
// SimRouter.
func (t *SimTransport) SendTo(addr []string, message *CodedMessage) error {
	if t.Closed {
		return errors.New("closed")
	}
	return t.Router.SendTo(addr, message)
}

// Receive a message from the receiving message queue.
func (t *SimTransport) Recv() (*CodedMessage, error) {
	if t.Closed {
		return nil, errors.New("closed")
	}
	if coded, ok := <-t.RecvCh; !ok {
		return nil, errors.New("closed")
	} else {
		return coded, nil
	}
}

// Close the simulated transport.
func (t *SimTransport) Close() error {
	if t.Closed {
		panic("closed")
	}
	t.Closed = true
	close(t.RecvCh)
	return nil
}
