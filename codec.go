package swim

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
