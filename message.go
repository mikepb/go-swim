package swim

// A message encapsulates pings, acknowledgements, and piggybacked broadcast
// messages sent between nodes. The events types are listed individually to
// simplify implementation. Codec implementations are assumed to omit empty
// fields for efficiency.
type Message struct {
	From       uint64          // The node sending this message
	To         uint64          // The node to which this message is intended
	Ping       PingEvent       // A ping for the recipient
	Ack        AckEvent        // A ping acknowledgement
	Requests   []RequestEvent  // Indirect ping requests
	Responses  []ResponseEvent // Indirect ping responses
	Alives     []AliveEvent    // Alive broadcasts
	Suspects   []SuspectEvent  // Suspect broadcasts
	Deaths     []DeathEvent    // Death broadcasts
	UserEvents []UserEvent     // User event broadcasts
}

// A coded message encapsulates a message for encoding and decoding.
type CodedMessage struct {
	Message *Message // The contained message, which may be nil
	Bytes   []byte   // The byte-encoded message
	Size    int      // The size of the message, if not byte-encoded
}

// Add an typed event to the message. If the event is a Ping or Ack, the
// original Ping or Ack is replaced if it exists.
func (m *Message) AddEvent(event interface{}) {
	switch event := event.(type) {
	case *PingEvent:
		m.Ping = *event

	case *AckEvent:
		m.Ack = *event

	case *RequestEvent:
		m.Requests = append(m.Requests, *event)

	case *ResponseEvent:
		m.Responses = append(m.Responses, *event)

	case *AliveEvent:
		m.Alives = append(m.Alives, *event)

	case *SuspectEvent:
		m.Suspects = append(m.Suspects, *event)

	case *DeathEvent:
		m.Deaths = append(m.Deaths, *event)

	case *UserEvent:
		m.UserEvents = append(m.UserEvents, *event)

	default:
		panic("unknown event type")
	}
}

// Count the number of events.
func (m *Message) EventCount() int {
	count := len(m.Requests) + len(m.Responses) + len(m.Alives) +
		len(m.Suspects) + len(m.Deaths) + len(m.UserEvents)
	if m.Ping != (PingEvent{}) {

	}
	if m.Ack != (AckEvent{}) {
		count += 1
	}
	return count
}
