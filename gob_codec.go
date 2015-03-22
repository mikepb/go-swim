package swim

import (
	"bytes"
	"encoding/gob"
)

// The gob codec uses Go's native gob encoding for encoding and decoding
// messages.
type GobCodec struct {
}

// Implementation of Codec.Decode()
func (c *GobCodec) Decode(coded *CodedMessage) error {
	msg := new(Message)

	// decode
	buf := bytes.NewReader(coded.Bytes)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(msg); err != nil {
		return err
	}

	// update values
	coded.Message = *msg

	return nil
}

// Implementation of Codec.Encode()
func (c *GobCodec) Encode(coded *CodedMessage) error {
	var buf bytes.Buffer

	// encode
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(coded.Message); err != nil {
		return err
	}

	// update values
	coded.Bytes = buf.Bytes()
	coded.Size = len(coded.Bytes)

	return nil
}

// Register interface{} values.
func init() {
	gob.Register(PingEvent{})
	gob.Register(AckEvent{})
	gob.Register(IndirectPingRequestEvent{})
	gob.Register(IndirectPingEvent{})
	gob.Register(IndirectAckEvent{})
	gob.Register(AntiEntropyEvent{})
	gob.Register(AliveEvent{})
	gob.Register(SuspectEvent{})
	gob.Register(DeathEvent{})
	gob.Register(UserEvent{})
}
