package swim

import (
	"math/rand"
	"time"
)

const kNetDelay = 5 * time.Millisecond
const kNetStdDev = 1 * time.Millisecond
const kMaxMessageLen = 512

// SimRouter routes messages between SimTransports for the network
// simulator.
type SimRouter struct {
	Routes        map[string]*SimTransport
	Rand          *rand.Rand
	NetDelay      time.Duration
	NetStdDev     time.Duration
	MaxMessageLen int
}

// Create a new SimRouter.
func NewSimRouter() *SimRouter {
	return &SimRouter{
		Routes:        make(map[string]*SimTransport),
		Rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		NetDelay:      kNetDelay,
		NetStdDev:     kNetStdDev,
		MaxMessageLen: kMaxMessageLen,
	}
}

// Create a new SimTransport associated with the given address. If a
// transport is already associated with the given address, the existing
// instance is returned.
func (r *SimRouter) NewTransport(addr string) *SimTransport {
	t, ok := r.Routes[addr]
	if !ok {
		t = NewSimTransport(r)
		r.Routes[addr] = t
	}
	return t
}

// Send a message to the first transport matching the addresses.
func (r *SimRouter) SendTo(addrs []string, message *CodedMessage) error {

	// delay the packet to simulate a "real" network
	time.AfterFunc(r.Delay(), func() {
		defer func() { recover() }()
		for _, addr := range addrs {
			if t, ok := r.Routes[addr]; ok {
				t.RecvCh <- message
				return
			}
		}
	})

	// silently fail to simulate UDP
	return nil
}

// Generate a normally distributed time delay with a mean of NetDelay and
// standard deviation of NetStdDev.
func (r *SimRouter) Delay() time.Duration {
	return time.Duration(r.Rand.NormFloat64()*float64(r.NetStdDev)) + r.NetDelay
}
