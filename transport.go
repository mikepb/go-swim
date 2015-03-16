package swim

// A transport implements the transport layer for the failure detector.
type Transport interface {

	// A hint of the maximum byte length of a message for this transport. This
	// value is used to limit the number of piggybacked broadcasts attached to
	// messages. If the maximum message length is smaller than the smallest
	// possible message length, the transport will receive outgoing messages
	// larger than the hinted maximum length. A negative number indicates
	// unlimited message length.
	MaxMessageLen() int

	// Send the given message to the node. The message is guaranteed to
	// already have been encoded by an encoder. The transport may support
	// sending the Go value directly, ignoring the encoded byte value.
	SendTo(node *Node, message *CodedMessage) error

	// The inbox channel receives messages from the network for processing.
	Inbox() <-chan *CodedMessage
}
