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

	// Send the given message to the addresses. The message is guaranteed to
	// already have been encoded by an encoder. The transport may support
	// sending the Go value directly, ignoring the encoded byte value. The
	// operation should timeout if the write deadline is set.
	SendTo(addr []string, message *CodedMessage) error

	// Receive a messages from the network, blocking until the next message
	// arrives. If the transport is closed, an appropriate error should be
	// returned. The operation should timeout if the read deadline is set.
	Recv() (*CodedMessage, error)

	// Close the transport.
	Close() error
}
