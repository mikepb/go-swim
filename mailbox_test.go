package swim

import (
	"errors"
	"testing"
)

func TestMailbox(t *testing.T) {
	mms := 928428
	codec := newTestCodec()
	transport := newTestTransport(mms)
	mbox := NewMailbox(codec, transport)

	// test sending
	node := &Node{Id: 12394}
	msg := &Message{From: 12394, To: 90210, Ack: AckEvent{9}}
	mbox.SendTo(node, msg)

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
	mbox.SendTo(node, msg)
	if len(mbox.Errors) == 0 || encodeError != <-mbox.Errors {
		t.Fatalf("Mailbox did not send error")
	} else if encoded := <-codec.encode; encoded.Message != msg {
		t.Fatalf("Mailbox did not provide codec with message")
	} else if len(transport.to) != 0 {
		t.Fatalf("Mailbox attempted to deliver message")
	} else if len(transport.outbox) != 0 {
		t.Fatalf("Mailbox attempted to deliver message")
	}

	// with transport error
	transportError := errors.New("transport error")
	transport.err <- transportError
	mbox.SendTo(node, msg)
	if encoded := <-codec.encode; encoded.Message != msg {
		t.Fatalf("Mailbox did not provide codec with message")
	} else if len(transport.to) == 0 {
		t.Fatalf("Mailbox did not attempt to deliver message")
	} else if len(transport.outbox) == 0 {
		t.Fatalf("Mailbox did not attempted to deliver message")
	} else if node != <-transport.to || encoded != <-transport.outbox {
		t.Fatalf("Mailbox attempted to deliver the wrong message")
	} else if len(mbox.Errors) == 0 || transportError != <-mbox.Errors {
		t.Fatalf("Mailbox did not send error")
	}

	// start receiving
	mbox.Start()

	// test receiving
	transport.inbox <- encoded
	if msg != <-mbox.Inbox {
		t.Fatalf("Mailbox did not receive message")
	} else if encoded != <-codec.decode {
		t.Fatalf("Mailbox did not decode message")
	}

	// test receiving with error
	decodeError := errors.New("decode error")
	codec.decodeErrs <- decodeError
	transport.inbox <- encoded
	if decodeError != <-mbox.Errors {
		t.Fatalf("Mailbox did not send error")
	} else if encoded != <-codec.decode {
		t.Fatalf("Mailbox did not decode message")
	}

	// stop receiving
	mbox.Stop()

	// test max message length
	if mlen := mbox.MaxMessageLen(); mlen != mms {
		t.Fatalf("Expected max message length %v got %v", mms, mlen)
	}
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

func (t *testTransport) Inbox() <-chan *CodedMessage {
	return t.inbox
}
