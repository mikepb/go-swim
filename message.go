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
	for _, event := range events {

		// remove indirection
		switch e := event.(type) {

		case *PingEvent:
			event = interface{}(*e)
		case *AckEvent:
			event = interface{}(*e)
		case *IndirectPingRequestEvent:
			event = interface{}(*e)
		case *IndirectPingEvent:
			event = interface{}(*e)
		case *IndirectAckEvent:
			event = interface{}(*e)
		case *AntiEntropyEvent:
			event = interface{}(*e)
		case *AliveEvent:
			event = interface{}(*e)
		case *SuspectEvent:
			event = interface{}(*e)
		case *DeathEvent:
			event = interface{}(*e)
		case *UserEvent:
			event = interface{}(*e)

		case PingEvent:
		case AckEvent:
		case IndirectPingRequestEvent:
		case IndirectPingEvent:
		case IndirectAckEvent:
		case AntiEntropyEvent:
		case AliveEvent:
		case SuspectEvent:
		case DeathEvent:
		case UserEvent:

		default:
			panic("invalid event")
		}

		// append direct value
		*m = append(*m, event)
	}
}

// Get the events in a message.
func (m Message) Events() []interface{} {
	return []interface{}(m)
}
