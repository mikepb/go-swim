package swim

import (
	"log"
	"time"
)

const kBufferSize = 8

type bcastSync struct {
	event BroadcastEvent
	done  chan struct{}
}

// Detector implements the SWIM failure detector. Remember to close the
// detector before discarding it to free resources.
type Detector struct {
	incarnation Seq
	broker      Broker

	// Node states in list form.
	nodes    SelectionList
	nodeMap  map[uint64]*InternalNode
	suspects map[uint64]*InternalNode

	// States for signaling the event loop.
	state      int
	started    bool
	stopping   chan bool
	stopped    chan bool
	events     chan interface{}
	broadcasts chan bcastSync

	// Private instance of the local node used for synchronization.
	localNode Node

	// The local node identifies this instance of the failure detector. The
	// node ID must be unique for all nodes. In addition, the node addresses
	// must be sufficiently distinct to allow messages to be routed to the
	// appropriate node ID. A transport may accept messages for multiple nodes
	// so long as it routes those messages appropriately. Changes to the local
	// node will not be picked up until the failure detector is restarted.
	LocalNode *Node

	// The number of direct probes to send per protocol period. The algorithm
	// described in the SWIM paper uses one direct probe per protocol period.
	DirectProbes uint

	// The number of indirect probes to send after a node fails to reply to
	// the direct probe. This is equivalent to the k parameter in the SWIM
	// paper.
	IndirectProbes uint

	// The probe interval controls the time between probing events. Longer
	// probe intervals will result in lower load on the network and slower
	// failure detection and broadcast rates, with shorter probe intervals
	// having the opposite effect. The probe interval must be at least three
	// times the longest expected probe timeout.
	ProbeInterval time.Duration

	// The probe timeout hints long to wait after a probe event before
	// requesting indirect probes from third-party nodes. The parameter sets
	// the initial node round-trip time, which is then automatically adjusted
	// for round-trip time delays in subsequent pings. The adjusted probe
	// timeouts are bounded from below by the value of this parameter and from
	// above by 1/3 of the probe interval.
	ProbeTimeout time.Duration

	// The retransmission multiplier controls how many times broadcast events
	// are retransmitted. The limit is calculated as
	//
	//     Retransmits = RetransmitMult * log(N+1)
	//
	// Broadcasts exceeding this limit are removed from the broadcast queue.
	RetransmitMult uint

	// The suspicion multiplier controls how long to wait until considering a
	// suspicious node as dead. The suspicion timeout is calculated as
	//
	//     SuspicionTimeout = SuspicionMult * log(N+1) * ProbeInterval
	//
	// Nodes marked as suspicious after this timeout are marked as dead.
	SuspicionMult uint

	// If not nil, log receipt of messages.
	Logger *log.Logger
}

// Start the failure detector.
func (d *Detector) Start() {

	// initialize
	if d.stopping == nil {

		// create channels
		d.stopping = make(chan bool, 1)
		d.stopped = make(chan bool, 1)
		d.events = make(chan interface{}, kBufferSize)
		d.broadcasts = make(chan bcastSync, 1)

		// create maps
		d.nodeMap = make(map[uint64]*InternalNode)
		d.suspects = make(map[uint64]*InternalNode)
	}

	// don't call multiple times!
	if d.started {
		panic("already started")
	}

	// flag as started
	d.state += 1
	d.started = true

	// update the private local node
	d.LocalNode.Incarnation.Witness(d.incarnation.Increment())
	d.localNode = *d.LocalNode

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

// Join the failure detection group by sending a join intent to the nodes
// represented by the given addresses.
func (d *Detector) Join(addrs ...[]string) {

	if !d.started {
		panic("not started")
	}

	// update the local node incarnation
	incarnation := d.reincarnation()

	// create alive event
	event := &AliveEvent{
		From: d.localNode.Id,
		Node: d.localNode,
	}

	// the incarnation is the only value that could change
	event.Node.Incarnation = incarnation

	// create the message
	msg := new(Message)
	msg.AddEvent(event)

	// send the event directly to the given addresses
	for _, addy := range addrs {
		d.broker.DirectTo(addy, msg)
	}
}

// Broadcast an intent to leave the group.
func (d *Detector) Leave() {

	if !d.started {
		panic("not started")
	}

	// create alive event
	event := &DeathEvent{
		From:        d.localNode.Id,
		Id:          d.localNode.Id,
		Incarnation: d.reincarnation(),
	}

	// broadcast it
	d.BroadcastSync(event)
}

// Broadcast an event asynchronously. If the detector is not running, the
// broadcast will be sent when the detector is started.
func (d *Detector) Broadcast(event BroadcastEvent) {
	d.broadcasts <- bcastSync{event: event}
}

// Broadcast an event and wait for the broadcast to be removed from the
// queue, either from invalidation or after reaching the broadcast
// transmission limit. If the detector is not running or there are no nodes
// other than the local node, the call will block indefinitely.
func (d *Detector) BroadcastSync(event BroadcastEvent) {
	done := make(chan struct{})
	d.broadcasts <- bcastSync{event, done}
	<-done
}

// Run the failure detector loop.
func (d *Detector) loop() {
	var probedNodes []*InternalNode
	var periodStartTime time.Time

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

		case bcast := <-d.broadcasts: // handle broadcasts
			if bcast.done == nil {
				d.broker.Broadcast(bcast.event)
			} else {
				d.broker.BroadcastSync(bcast.event, bcast.done)
			}

		case event := <-d.events: // handle events
			d.handle(periodStartTime, event)

		case <-timer.C: // probe timeout
			if !periodStartTime.IsZero() && probedNodes != nil {
				d.indirectProbe(periodStartTime, probedNodes)
			}

		case t := <-ticker.C: // protocol period

			// handle suspicion from the previous protocol period
			if !periodStartTime.IsZero() && probedNodes != nil {
				d.suspected(periodStartTime, probedNodes)
			}
			periodStartTime = t

			// send out the probes
			probedNodes = d.probe()

			// set the timer for maybe sending indirect probes
			timer.Reset(d.boundedTimeout(probedNodes))
		}
	}
}

// Send fresh probes.
func (d *Detector) probe() (nodes []*InternalNode) {
	max := d.nodes.Len()
	if int(d.DirectProbes) < max {
		max = int(d.DirectProbes)
	}
	for i := 0; i < max; i += 1 {
		if node := d.nodes.Next(); node != nil {
			d.sendTo(node, d.ping())
			nodes = append(nodes, node)
		}
	}
	return
}

// Send indirect probes.
func (d *Detector) indirectProbe(periodStartTime time.Time, nodes []*InternalNode) {

	// batch requests for the indirect probes
	requests := []interface{}{}
	for _, node := range nodes {
		if node.LastAckTime.Before(periodStartTime) {
			requests = append(requests, d.pingRequest(node))
		}
	}

	// send the indirect probe requests
	max := d.nodes.Len()
	if int(d.IndirectProbes) < max {
		max = int(d.IndirectProbes)
	}
	for i := 0; i < max; i += 1 {
		if node := d.nodes.Next(); node != nil {
			d.sendTo(node, requests...)
		}
	}
}

// Send suspect events.
func (d *Detector) suspected(periodStartTime time.Time, nodes []*InternalNode) {
	for _, node := range nodes {
		if node.LastAckTime.Before(periodStartTime) {

			// these nodes have not responded
			d.Broadcast(d.suspect(node))

			// save into global suspect list
			d.suspects[node.Id] = node
		}
	}

	// nodes that have not disputed their suspect status before this time are
	// considered dead
	deathTime := time.Now().Add(-d.suspicionTime())

	// determine which nodes have died
	for id, node := range d.suspects {

		if node.State != Suspect {

			// if the node is not suspect, then it's alive or dead (as a cat)
			delete(d.suspects, id)

		} else if node.LastAckTime.Before(deathTime) {

			// the node is dead if it hasn't disputed its suspicion
			node.State = Dead
			node.Incarnation.Increment()
			d.Broadcast(d.death(node))
			delete(d.suspects, id)

		} else {

			// otherwise, the node is still suspect

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
			if d.Logger != nil {
				d.Logger.Printf("[swim:detector:recv] %v", err)
			}
			continue
		}

		// log the message
		if d.Logger != nil {
			d.Logger.Flags()
			d.Logger.Printf("[swim:detector:recv] %v", msg)
		}

		// queue events
		for _, event := range msg.Events() {
			d.events <- event
		}
	}
}

func (d *Detector) handle(lastTick time.Time, event interface{}) {
	switch event := event.(type) {
	case *PingEvent:
		d.handlePing(event)

	case *AckEvent:
		d.handleAck(lastTick, event)

	case *IndirectPingRequestEvent:
		d.handleIndirectPingRequest(event)

	case *IndirectPingEvent:
		d.handleIndirectPing(event)

	case *IndirectAckEvent:
		d.handleIndirectAck(lastTick, event)

	case *AntiEntropyEvent:
		d.handleAntiEntropy(event)

	case *AliveEvent:
		d.handleAlive(event)

	case *SuspectEvent:
		d.handleSuspect(event)

	case *DeathEvent:
		d.handleDeath(event)

	case *UserEvent:
		d.handleUserEvent(event)
	}
}

// Handle pings.
func (d *Detector) handlePing(event *PingEvent) {

	// lookup the node
	node, events, ok := d.lookup(event.From, event.Incarnation, nil)

	// can't acknowledge without return address
	if !ok || node.Addrs == nil {
		return
	}

	// acknowledge the ping
	events = append(events, d.ack(event.Time))
	d.sendTo(node, events...)
}

// Handle indirect ping requests.
func (d *Detector) handleIndirectPingRequest(event *IndirectPingRequestEvent) {

	// lookup the nodes
	from, fromEvents, _ := d.lookup(event.From, event.Incarnation, event.Addrs)
	target, targetEvents, _ := d.lookup(event.Target, event.TargetIncarnation, event.TargetAddrs)

	// send indirect ping
	fromEvents = append(fromEvents, d.pingVia(from, event.Time))
	d.sendTo(target, targetEvents...)

	// send anti-entropy response to requestor
	if fromEvents != nil {
		d.sendTo(target, fromEvents...)
	}
}

// Handle indirect pings.
func (d *Detector) handleIndirectPing(event *IndirectPingEvent) {

	// lookup the nodes
	node, nodeEvents, _ := d.lookup(event.From, event.Incarnation, event.Addrs)
	via, viaEvents, _ := d.lookup(event.Via, event.ViaIncarnation, event.ViaAddrs)

	// send indirect ack
	events := append(nodeEvents, viaEvents...)
	events = append(events, d.indirectAck(event.Time, via, event.ViaTime))
	d.sendTo(node, events...)
}

// Handle acknowledgements.
func (d *Detector) handleAck(lastTick time.Time, event *AckEvent) {

	// lookup the node
	node, events, _ := d.lookup(event.From, event.Incarnation, nil)

	// check the timestamp
	if lastTick.IsZero() {
		// no-op
	} else if event.Time.IsZero() || event.Time.Before(lastTick) {
		// ignore if invalid time or very late response
		return
	} else if event.Time.Before(node.LastAckTime) {
		// node already acknowledged before now
		return
	}

	// set last ack time
	node.LastAckTime = time.Now()

	// update RTT; this extends the RTT in the case of an indirect ack to
	// reduce the likelihood of future false negatives from slow nodes
	node.RTT.Update(time.Since(event.Time))

	// send alive message if node isn't marked as alive
	if node.State != Alive {
		d.Broadcast(d.alive(node))
	}

	// send anti-entropy response
	if events != nil {
		d.sendTo(node, events...)
	}
}

// Handle indirect acknowledgement.
func (d *Detector) handleIndirectAck(lastTick time.Time, event *IndirectAckEvent) {

	// handle the ack locally
	d.handleAck(lastTick, &event.AckEvent)

	// lookup the node
	node, events, _ := d.lookup(event.Via, 0, nil)

	// update ack event for requesting node
	ack := &event.AckEvent
	ack.Time = event.ViaTime

	// relay ack to requestor
	events = append(events, ack)
	d.sendTo(node, events...)
}

// Handle anti-entropy event.
func (d *Detector) handleAntiEntropy(event *AntiEntropyEvent) {

	// lookup the node
	node, _, _ := d.lookup(event.Id, 0, nil)

	// update the node
	node.Node = event.Node
	node.State = event.State
}

// Handle alive event.
func (d *Detector) handleAlive(event *AliveEvent) {

	// handle the common actions
	node := d.handleStateBroadcast(event, event.From, event.Id, event.Incarnation)
	if node == nil {
		return
	}

	// update local instance
	node.Node = event.Node
	node.State = Alive
}

// Handle suspect event.
func (d *Detector) handleSuspect(event *SuspectEvent) {

	// handle the common actions
	node := d.handleStateBroadcast(event, event.From, event.Id, event.Incarnation)
	if node == nil {
		return
	}

	// update local instance
	node.State = Suspect
}

// Handle death event.
func (d *Detector) handleDeath(event *DeathEvent) {

	// handle the common actions
	node := d.handleStateBroadcast(event, event.From, event.Id, event.Incarnation)
	if node == nil {
		return
	}

	// update local instance
	node.State = Dead
}

// Handle the alive, suspect, and death state broadcasts.
func (d *Detector) handleStateBroadcast(event BroadcastEvent, from, id uint64, incarnation Seq) *InternalNode {

	// lookup the node
	node, events, _ := d.lookup(id, 0, nil)

	// ignore message if already seen
	if node.Incarnation.Compare(incarnation) >= 0 {
		return nil
	}

	// update incarnation number
	node.Incarnation.Witness(incarnation)

	// re-broadcast
	d.Broadcast(event)

	// handle anti-entropy if we can reach the original node
	if events != nil {
		fromNode, _, _ := d.lookup(from, 0, nil)
		if fromNode.Addrs != nil {
			d.sendTo(fromNode, events...)
		}
	}

	return node
}

// Handle user event.
func (d *Detector) handleUserEvent(event *UserEvent) {

	panic("not implemented")

	// ignore message if already seen
	// TODO: figure out how to do this elegantly

	// re-broadcast
	d.Broadcast(event)
}

// Ping the node.
func (d *Detector) ping() *PingEvent {
	return &PingEvent{
		From:        d.localNode.Id,
		Incarnation: d.localNode.Incarnation.Get(),
		Time:        time.Now(),
	}
}

// Acknowledge a ping.
func (d *Detector) ack(t time.Time) *AckEvent {
	return &AckEvent{
		From:        d.localNode.Id,
		Incarnation: d.localNode.Incarnation.Get(),
		Time:        t,
	}
}

// Send an indirect ping request.
func (d *Detector) pingRequest(node *InternalNode) *IndirectPingRequestEvent {
	return &IndirectPingRequestEvent{
		From:        d.localNode.Id,
		Incarnation: d.localNode.Incarnation.Get(),
		Addrs:       d.localNode.Addrs,
		Time:        time.Now(),
		Target:      node.Id,
		TargetAddrs: node.Addrs,
	}
}

// Indirectly ping a node.
func (d *Detector) pingVia(via *InternalNode, viaTime time.Time) *IndirectPingEvent {
	return &IndirectPingEvent{
		PingEvent: *d.ping(),
		Addrs:     d.localNode.Addrs,
		Via:       via.Id,
		ViaAddrs:  via.Addrs,
		ViaTime:   viaTime,
	}
}

// Indirectly ack a node.
func (d *Detector) indirectAck(t time.Time, via *InternalNode, viaTime time.Time) *IndirectAckEvent {
	return &IndirectAckEvent{
		AckEvent: *d.ack(t),
		Via:      via.Id,
		ViaTime:  viaTime,
	}
}

// Broadcast news that a node is alive.
func (d *Detector) alive(node *InternalNode) *AliveEvent {
	node.Incarnation.Increment()
	return &AliveEvent{
		From: d.localNode.Id,
		Node: node.Node,
	}
}

// Broadcast news that a node is suspected of failure.
func (d *Detector) suspect(node *InternalNode) *SuspectEvent {
	return &SuspectEvent{
		From:        d.localNode.Id,
		Id:          node.Id,
		Incarnation: d.incarnation.Increment(),
	}
}

// Broadcast news that a node has died.
func (d *Detector) death(node *InternalNode) *DeathEvent {
	return &DeathEvent{
		From:        d.localNode.Id,
		Id:          node.Id,
		Incarnation: d.incarnation.Increment(),
	}
}

// Send an anti-entropy event.
func (d *Detector) antiEntropy(node *InternalNode) *AntiEntropyEvent {
	return &AntiEntropyEvent{
		From:  d.localNode.Id,
		Node:  node.Node,
		State: node.State,
	}
}

// Send events to a node.
func (d *Detector) sendTo(node *InternalNode, events ...interface{}) {
	msg := new(Message)
	msg.AddEvent(events...)
	d.broker.SendTo(node.Node.Addrs, msg)
	d.pruneBroadcasts()
}

// Prune broadcasts that have been transmitted enough times.
func (d *Detector) pruneBroadcasts() {
	limit := d.retransmitLimit()
	d.broker.Broadcasts.Prune(func(b *Broadcast) bool {
		return b.Attempts >= limit
	})
}

// Get the singleton node for the given ID.
func (d *Detector) lookup(id uint64, incarnation Seq, addrs []string) (*InternalNode, []interface{}, bool) {

	// lookup the node
	node, ok := d.nodeMap[id]
	if !ok {

		// create node
		node = &InternalNode{
			Node: Node{
				Id:    id,
				Addrs: addrs,
			},
		}

		// hint RTT
		node.RTT.Hint(d.ProbeTimeout)

		// save node
		d.nodeMap[id] = node

		// insert into selection list
		d.nodes.Add(node)
	}

	// witness the global incarnation number
	d.incarnation.Witness(incarnation)

	// anti-entropy
	events := []interface{}{}
	if node.Incarnation.Compare(incarnation) > 0 {
		events = append(events, d.antiEntropy(node))
	}

	// return the singleton
	return node, events, !ok
}

// Increment the local node incarnation number.
func (d *Detector) reincarnation() Seq {
	incarnation := d.localNode.Incarnation.Witness(d.incarnation.Increment())
	d.LocalNode.Incarnation.Witness(incarnation)
	return incarnation
}

func (d *Detector) boundedTimeout(nodes []*InternalNode) time.Duration {

	// get max timeout in node set
	timeout := time.Duration(d.ProbeTimeout)
	for _, node := range nodes {
		if rtt := node.RTT.Get(); timeout < rtt {
			timeout = rtt
		}
	}

	// bound timeout to 1/3 protocol period
	if max := d.ProbeInterval / 3; timeout > max {
		timeout = max
	}

	return timeout
}

func (d *Detector) suspicionTime() time.Duration {
	// the suspicion time is calculated as mult*log(N+1); division by three is
	// to convert from log base 2 to base 10 (approximately)
	return (time.Duration(d.SuspicionMult) *
		time.Duration(log2ceil(d.nodes.Len()+1)) * d.ProbeInterval / 3)
}

func (d *Detector) retransmitLimit() uint {
	// calculate the retransmission limit as mult*log(N+1); the division by three
	return d.RetransmitMult * uint(log2ceil(d.nodes.Len()+1)) / 3
}
