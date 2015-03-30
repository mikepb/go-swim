package swim

import (
	"math/rand"
	"runtime"
	"sync"
	"time"
)

const kNetDelay = 5 * time.Millisecond
const kNetStdDev = kNetDelay / 10
const kMaxMessageLen = 512

// SimRouter routes messages between SimTransports for the network
// simulator.
type SimRouter struct {
	Routes        map[string]*SimTransport
	Rand          *rand.Rand
	NetDelay      time.Duration
	NetStdDev     time.Duration
	MaxMessageLen int
	l             sync.Mutex
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

	deliver := func() {
		defer func() { recover() }()
		for _, addr := range addrs {
			if t, ok := r.Routes[addr]; ok {
				t.RecvCh <- message
			}
		}
	}

	delay := r.Delay()

	// support no delay
	if delay == 0 {
		go deliver()
		return nil
	}

	defer runtime.Gosched()

	// delay the packet to simulate a "real" network
	time.AfterFunc(delay, deliver)

	// silently fail to simulate UDP
	return nil
}

// Generate a normally distributed time delay with a mean of NetDelay and
// standard deviation of NetStdDev.
func (r *SimRouter) Delay() time.Duration {

	// rand is not concurrent
	r.l.Lock()
	n := r.Rand.NormFloat64()
	r.l.Unlock()

	d := time.Duration(n*float64(r.NetStdDev)) + r.NetDelay
	if d < 0 {
		return 0
	}
	return d
}
