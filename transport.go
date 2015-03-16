package swim

// A transport implements the transport layer for the failure detector.
type Transport interface {

	// A hint of the maximum size of a message for this transport. This value
	// is used to limit the number of piggybacked broadcasts attached to
	// messages. If the maximum message size is smaller than the smallest
	// possible message size, the transport will receive outgoing messages
	// larger than the hinted maximum size. A negative number indicates
	// unlimited message size.
	MaxMessageSize() int

	// Send the given message to the node. The message is guaranteed to
	// already have been encoded by an encoder. The transport may support
	// sending the Go value directly, ignoring the encoded byte value.
	Send(node *Node, message *CodedMessage) error

	// The inbox channel receives messages from the network for processing.
	Inbox() <-chan *CodedMessage
}
