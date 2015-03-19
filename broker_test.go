package swim

import (
	"errors"
	"testing"
	"time"
)

func TestMailbox(t *testing.T) {
	mms := 512
	codec := newTestCodec()
	transport := newTestTransport(mms)
	broker := &Broker{Transport: transport, Codec: codec}

	// test sending
	node := &Node{Id: 12394}
	msg := &Message{From: 12394, To: 90210}
	msg.AddEvent(&AckEvent{From: 12394, Timestamp: time.Unix(0, 9)})
	broker.DirectTo(node, msg)

	// should have encoded message
	if len(codec.encode) == 0 {
		t.Fatalf("Mailbox did not encode message")
	}
	encoded := <-codec.encode
	if encoded.Message != msg {
		t.Fatalf("Mailbox did not provide codec with message")
	}

	// should have sent the message
	if len(transport.to) == 0 || node != <-transport.to {
		t.Fatalf("Mailbox did not deliver message")
	} else if len(transport.outbox) == 0 || encoded != <-transport.outbox {
		t.Fatalf("Mailbox did not use encoded message")
	}

	// with codec error
	encodeError := errors.New("encode error")
	codec.encodeErrs <- encodeError
	if err := broker.DirectTo(node, msg); err != encodeError {
		t.Fatalf("Mailbox did not send error")
	} else if encoded := <-codec.encode; encoded.Message != msg {
		t.Fatalf("Mailbox did not provide codec with message")
	} else if len(transport.to) != 0 || len(transport.outbox) != 0 {
		t.Fatalf("Mailbox attempted to deliver message")
	}

	// with transport error
	transportError := errors.New("transport error")
	transport.err <- transportError
	if err := broker.DirectTo(node, msg); err != transportError {
		t.Fatalf("Mailbox did not provide transport with message")
	} else if len(transport.to) == 0 {
		t.Fatalf("Mailbox did not attempt to deliver message")
	} else if len(transport.outbox) == 0 {
		t.Fatalf("Mailbox did not attempted to deliver message")
	} else if node != <-transport.to || <-codec.encode != <-transport.outbox {
		t.Fatalf("Mailbox attempted to deliver the wrong message")
	}

	// test receiving
	transport.inbox <- encoded
	if m, err := broker.Recv(); err != nil || m != msg {
		t.Fatalf("Mailbox did not receive message")
	} else if encoded != <-codec.decode {
		t.Fatalf("Mailbox did not decode message")
	}

	// test receiving with error
	decodeError := errors.New("decode error")
	codec.decodeErrs <- decodeError
	transport.inbox <- encoded
	if _, err := broker.Recv(); err != decodeError {
		t.Fatalf("Mailbox did not send error")
	} else if encoded != <-codec.decode {
		t.Fatalf("Mailbox did not decode message")
	}

	// TODO: test with broadcasts
	// TODO: test adding broadcasts
	// TODO: test adding broadcasts sync
}

type testCodec struct {
	decode     chan *CodedMessage
	encode     chan *CodedMessage
	decodeErrs chan error
	encodeErrs chan error
}

func newTestCodec() *testCodec {
	return &testCodec{
		decode:     make(chan *CodedMessage, 1),
		encode:     make(chan *CodedMessage, 1),
		decodeErrs: make(chan error, 1),
		encodeErrs: make(chan error, 1),
	}
}

func (c *testCodec) Decode(message *CodedMessage) error {
	c.decode <- message
	if len(c.decodeErrs) > 0 {
		return <-c.decodeErrs
	}
	return nil
}

func (c *testCodec) Encode(message *CodedMessage) error {
	c.encode <- message
	if len(c.encodeErrs) > 0 {
		return <-c.encodeErrs
	}
	message.Size = 16
	return nil

}

type testTransport struct {
	mms    int
	inbox  chan *CodedMessage
	outbox chan *CodedMessage
	to     chan *Node
	err    chan error
}

func newTestTransport(mms int) *testTransport {
	return &testTransport{
		mms:    mms,
		inbox:  make(chan *CodedMessage, 1),
		outbox: make(chan *CodedMessage, 1),
		to:     make(chan *Node, 1),
		err:    make(chan error, 1),
	}
}

func (t *testTransport) MaxMessageLen() int {
	return t.mms
}

func (t *testTransport) SendTo(node *Node, message *CodedMessage) error {
	t.to <- node
	t.outbox <- message
	if len(t.err) > 0 {
		return <-t.err
	}
	return nil
}

func (t *testTransport) Recv() (msg *CodedMessage, err error) {
	if len(t.err) > 0 {
		err = <-t.err
	}
	msg = <-t.inbox
	return
}

func (t *testTransport) SetDeadline(d time.Time) error {
	return nil
}

func (t *testTransport) SetReadDeadline(d time.Time) error {
	return nil
}

func (t *testTransport) SetWriteDeadline(d time.Time) error {
	return nil
}

func (t *testTransport) Close() error {
	return nil
}
