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
