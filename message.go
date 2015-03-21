package swim

// A message contains a set of events.
type Message []interface{}

// A coded message encapsulates a message for encoding and decoding.
type CodedMessage struct {
	Message *Message // The contained message, which may be nil
	Bytes   []byte   // The byte-encoded message
	Size    int      // The size of the message, if not byte-encoded
}

// Add an typed event to the message.
func (m *Message) AddEvent(events ...interface{}) {
	*m = append(*m, events...)
}

// Get the events in a message.
func (m Message) Events() []interface{} {
	return []interface{}(m)
}
