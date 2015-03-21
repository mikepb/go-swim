package swim

import (
	"log"
	"time"
)

const kBufferSize = 8
const kProbeCount = 1

// Detector implements the SWIM failure detector. Remember to close the
// detector before discarding it to free resources.
type Detector struct {
	sequence    Seq
	incarnation Seq

	broker Broker

	localNode *InternalNode
	nodes     SelectionList
	nodeMap   map[uint64]*InternalNode

	// States for signaling the event loop.
	state    int
	started  bool
	stopping chan bool
	stopped  chan bool
	events   chan interface{}

	ProbeInterval  time.Duration
	ProbeTimeout   time.Duration
	IndirectProbes uint
	RegionCount    uint // Number of regions to create
	Logger         *log.Logger
}

// Start the failure detector.
func (d *Detector) Start() {

	// initialize
	if d.stopping == nil {

		// create channels
		d.stopping = make(chan bool, 1)
		d.stopped = make(chan bool, 1)
		d.events = make(chan interface{}, kBufferSize)
	}

	// don't call multiple times!
	if d.started {
		panic("already started")
	}

	// flag as started
	d.state += 1
	d.started = true

	// receive messages asynchronously
	go d.recv()

	// run everything in a single goroutine event loop to avoid locks
	// (except for channel locks)
	go d.loop()
}

// Stop the failure detector.
func (d *Detector) Stop() {

	// don't call multiple times!
	if !d.started {
		panic("not started")
	}

	// signal goroutine to stop
	d.stopping <- true

	// receive acknowledgement
	<-d.stopped

	// the message receiver won't stop until a message is received...
	d.started = false
}

// Stop the failure detector and close the underlying transport.
func (d *Detector) Close() error {

	// stop if running
	if d.started {
		d.Stop()
	}

	// close the broker
	return d.broker.Close()
}

// Broadcast an event asynchronously. If the detector is not running, the
// broadcast will be sent when the detector is started.
func (d *Detector) Broadcast(msg BroadcastEvent) {
	d.broker.Broadcast(msg)
}

// Broadcast an event and wait for the broadcast to be removed from the
// queue, either from invalidation or after reaching the broadcast
// transmission limit. If the detector is not running or there are no nodes
// other than the local node, the call will block indefinitely.
func (d *Detector) BroadcastSync(msg BroadcastEvent) {
	d.broker.BroadcastSync(msg)
}

// Run the failure detector loop.
func (d *Detector) loop() {
	var nodes []*InternalNode
	var lastTick time.Time

	ticker := time.NewTicker(d.ProbeInterval)
	defer ticker.Stop()

	timer := time.NewTimer(0)
	timer.Stop()
	defer timer.Stop()

	for {
		select {
		case <-d.stopping: // stop signal
			d.stopped <- false
			return

		case event := <-d.events: // process events
			d.handle(event)

		case <-timer.C: // probe timeout
			if !lastTick.IsZero() && nodes != nil {
				d.indirectProbe(lastTick, nodes)
			}

		case t := <-ticker.C: // protocol period
			if !lastTick.IsZero() && nodes != nil {
				d.suspect(lastTick, nodes)
			}
			nodes = d.probe(t)
			timer.Reset(d.ProbeTimeout)
			lastTick = t
		}
	}
}

// Send fresh probes.
func (d *Detector) probe(t time.Time) (nodes []*InternalNode) {
	for i := 0; i < kProbeCount && i < d.nodes.Len(); i += 1 {
		if node := d.nodes.Next(); node != nil {
			d.sendTo(&node.Node, &PingEvent{
				From:        d.localNode.Id,
				Addrs:       d.localNode.Addrs,
				Incarnation: d.localNode.Incarnation,
				Time:        t,
			})
			nodes = append(nodes, node)
		}
	}
	return
}

// Send indirect probes.
func (d *Detector) indirectProbe(t time.Time, nodes []*InternalNode) {
	for _, target := range nodes {
		for i := 0; i < int(d.IndirectProbes) && i < d.nodes.Len(); i += 1 {
			if node := d.nodes.Next(); node != nil && node.LastAckTime.Before(t) {
				d.sendTo(&node.Node, &IndirectPingRequestEvent{
					From:  d.localNode.Id,
					Id:    target.Id,
					Addrs: target.Addrs,
				})
			}
		}
	}
}

// Send suspect events.
func (d *Detector) suspect(t time.Time, nodes []*InternalNode) {
	for _, target := range nodes {
		for i := 0; i < int(d.IndirectProbes) && i < d.nodes.Len(); i += 1 {
			if node := d.nodes.Next(); node != nil && node.LastAckTime.Before(t) {
				d.broker.Broadcast(&SuspectEvent{
					From:        d.localNode.Id,
					Id:          target.Id,
					Incarnation: target.Incarnation.Increment(),
				})
			}
		}
	}
}

// Receive messages from the network.
func (d *Detector) recv() {

	// loop while we're active
	// the state prevents multiple concurrent goroutines
	for state := d.state; d.started && d.state == state; {

		// receive message from broker
		msg, err := d.broker.Recv()
		if err != nil {
			d.Logger.Printf("[swim:detector:recv] %v", err)
		}

		// queue events
		if msg.To == d.localNode.Id {
			for _, event := range msg.Events {
				d.events <- event
			}
		}
	}
}

func (d *Detector) handle(event interface{}) {
	switch event := event.(type) {
	case *PingEvent:
	case *AckEvent:
	case *IndirectPingEvent:
	case *IndirectAckEvent:
	case *AliveEvent:
	case *SuspectEvent:
	case *DeathEvent:
	case *UserEvent:
		_ = event
	}
}

func (d *Detector) sendTo(node *Node, event interface{}) {
	msg := &Message{From: d.localNode.Id, To: node.Id}
	msg.AddEvent(event)
	d.broker.SendTo(node, msg)
}
