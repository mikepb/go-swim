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

// A coded message encapsulates a
type CodedMessage struct {
	Message
	Bytes []byte
	Size  int
}

// A codec handles message encoding as sent over the transport
// implementation.
type Codec interface {

	// Decode the given message into the Message field. The data referenced by
	// the CodedMessage instance is guaranteed to not have changed between
	// calls with the same instance of CodedMessage.
	Decode(message *CodedMessage) error

	// Encode the given message from the Message field. The Message is
	// guaranteed to be the same between calls with the same instance of
	// CodedMessage.
	Encode(message *CodedMessage) error
}
