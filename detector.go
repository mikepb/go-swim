package swim

import (
	"log"
	"net"
	"time"
)

const kBufferSize = 8
const kProbeCount = 1

type Detector struct {
	sequence    Seq
	incarnation Seq

	broker Broker

	localNode *InternalNode
	nodes     SelectionList
	nodeMap   map[uint64]*InternalNode

	state chan bool
	inbox chan interface{}

	ProbeInterval  time.Duration
	ProbeTimeout   time.Duration
	IndirectProbes uint
	RegionCount    uint // Number of regions to create
	Logger         *log.Logger
}

func NewDetector() *Detector {
	d := &Detector{
		state: make(chan bool, 1),
		inbox: make(chan interface{}, kBufferSize),
	}

	// empty = running
	// true  = stop signal
	// false = stopped
	d.state <- false

	return d
}

// Broadcast an event and wait for the broadcast to be removed from the
// queue, either from invalidation or after reaching the broadcast
// transmission limit.
func (d *Detector) Broadcast(msg BroadcastEvent) {
	d.broker.BroadcastSync(msg)
}

// Start the failure detector.
func (d *Detector) Start() {

	// don't call multiple times!
	if len(d.state) == 0 || <-d.state {
		panic("already started")
	}

	// run everything in a single goroutine event loop to avoid locks
	// (except for channel locks)
	go d.loop()
}

// Stop the failure detector.
func (d *Detector) Stop() {

	// don't call multiple times!
	if len(d.state) != 0 {
		panic("already stopped")
	}

	// signal goroutine to stop
	d.state <- true

	// receive acknowledgement
	d.state <- <-d.state
}

// Stop the failure detector and close the underlying transport.
func (d *Detector) Close() error {

	// stop if running
	if len(d.state) == 0 {
		d.Stop()
	}

	// close the broker
	return d.broker.Close()
}

// Run the failure detector loop.
func (d *Detector) loop() {
	var nodes []*InternalNode
	var lastTick time.Time

	ticker := time.NewTicker(d.ProbeInterval)
	timer := time.NewTimer(0)
	timer.Stop()

	for {
		select {
		case <-d.state: // stop signal
			d.state <- false
			return

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

		default: // read from network
			d.recv()
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
				Timestamp:   t,
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

	// set deadline
	t := time.Now().Add(time.Millisecond)
	if err := d.broker.Transport.SetReadDeadline(t); err != nil {
		d.Logger.Printf("[swim:detector:recv] %v", err)
		return
	}

	// receive message from broker
	msg, err := d.broker.Recv()
	if err != nil {
		switch err := err.(type) {
		case net.Error:
			if !err.Timeout() {
				d.Logger.Printf("[swim:detector:recv] %v", err)
			}
		default:
			d.Logger.Printf("[swim:detector:recv] %v", err)
		}
		return
	}

	// handle the events contained in the message
	if msg.To == d.localNode.Id {
		for _, event := range msg.Events {
			d.handle(event)
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
		d.inbox <- event
	}
}

func (d *Detector) sendTo(node *Node, event interface{}) {
	msg := &Message{From: d.localNode.Id, To: node.Id}
	msg.AddEvent(event)
	d.broker.SendTo(node, msg)
}
