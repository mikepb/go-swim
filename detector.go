package swim

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const kBufferSize = 8

// Detector implements the SWIM failure detector. Remember to close the
// detector before discarding it to free resources.
type Detector struct {

	// The global incarnation number used for broadcasting node state updates.
	incarnation Seq

	// The Broker instance used for sending and receiving network messages.
	broker *Broker

	// Node states.
	nodes       SelectionList
	nodeMap     map[uint64]*InternalNode
	actives     map[uint64]bool
	activeList  []Node
	activeCount int64
	suspects    map[uint64]*InternalNode

	// States for signaling the event loop.
	state    int
	started  bool
	stopping chan struct{}
	stopped  chan struct{}
	joins    chan []string
	period   time.Time

	// Concurrency control.
	l sync.Mutex

	// The local node identifies this instance of the failure detector. The
	// node ID must be unique for all nodes. In addition, the node addresses
	// must be sufficiently distinct to allow messages to be routed to the
	// appropriate node ID. A transport may accept messages for multiple nodes
	// so long as it routes those messages appropriately. The local node must
	// not be accessed when the Detector is running.
	LocalNode Node

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

	// The Transport implementation to use. The instance must not be accessed
	// outside the detector.
	Transport Transport

	// The Codec implementation to use. The instance must not be accessed
	// outside the detector.
	Codec Codec

	// The SelectionList implementation to use. The instance must not be
	// accessed outside the detector.
	SelectionList SelectionList

	// If not nil, log receipt of messages.
	Logger *log.Logger

	// If not nil, channel on which to send nodes when they are updated.
	UpdateCh chan Node

	// If not nil, channel on which to send messages received by this node.
	MessageCh chan Message
}

// Start the failure detector.
func (d *Detector) Start() {

	// initialize
	if d.stopping == nil {

		// create broker
		d.broker = NewBroker(d.Transport, d.Codec)

		// save selection list
		d.nodes = d.SelectionList

		// create channels
		d.stopping = make(chan struct{}, 1)
		d.stopped = make(chan struct{}, 1)
		d.joins = make(chan []string, 1)

		// create maps
		d.nodeMap = make(map[uint64]*InternalNode)
		d.actives = make(map[uint64]bool)
		d.suspects = make(map[uint64]*InternalNode)
	}

	// don't call multiple times!
	if d.started {
		panic("already started")
	}

	// flag as started
	d.state += 1
	d.started = true

	// update local node state
	d.LocalNode.State = Alive
	d.LocalNode.Incarnation.Witness(d.incarnation.Increment())

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
	d.state += 1
	d.stopping <- struct{}{}

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
// represented by the given addresses. The detector is started if not
// already running.
func (d *Detector) Join(addrs ...string) {

	if !d.started {
		d.Start()
	}

	// create alive event
	d.l.Lock()
	event := d.aliveNode(&d.LocalNode)
	localAddrs := d.LocalNode.Addrs
	d.l.Unlock()

	// broadcast death event to invalidate old broadcasts
	d.Broadcast(event)

	// create the message
	msg := new(Message)
	msg.AddEvent(event)

	// don't send to self
	ignore := make(map[string]bool)
	for _, addr := range localAddrs {
		ignore[addr] = true
	}

	// send the event directly to the given addresses
	for _, addy := range addrs {
		if !ignore[addy] {
			d.broker.DirectTo([]string{addy}, msg)
			ignore[addy] = true
		}
	}
}

// Broadcast an intent to leave the group. The detector is stopped if not
// already stopped.
func (d *Detector) Leave() {

	// stop if running
	if d.started {
		d.Stop()
	}

	// we're dead
	d.LocalNode.Incarnation.Witness(d.incarnation.Increment())
	d.LocalNode.State = Dead

	// broadcast death event to invalidate old broadcasts
	event := d.deathNode(&d.LocalNode)
	d.Broadcast(event)

	// prepare death message
	msg := &Message{From: d.LocalNode.Id}
	msg.AddEvent(event)

	// send the death broadcast
	nodes := d.nodes.List()
	for i, n, m := 0, len(nodes), int(d.IndirectProbes); i < n && i < m; i += 1 {
		node := nodes[i]
		d.broker.DirectTo(node.Addrs, msg)
	}
}

// Broadcast an event asynchronously. If the detector is not running, the
// broadcast will be sent when the detector is started.
func (d *Detector) Broadcast(event BroadcastEvent) {
	d.broker.Broadcast(event)
}

// Broadcast an event and wait for the broadcast to be removed from the
// queue, either from invalidation or after reaching the broadcast
// transmission limit. If the detector is not running or there are no nodes
// other than the local node, the call will block indefinitely.
func (d *Detector) BroadcastSync(event BroadcastEvent) {
	<-d.broker.BroadcastSync(event)
}

// Retrieve a list of member nodes that have not been marked as dead. The
// returned list should not be modified.
func (d *Detector) Members() []Node {
	d.l.Lock()
	defer d.l.Unlock()

	// send cached
	if d.activeList != nil || d.ActiveCount() == 0 {
		return d.activeList
	}

	// make new list
	nodes := make([]Node, 0, d.ActiveCount())
	for id := range d.actives {
		nodes = append(nodes, d.nodeMap[id].Node)
	}
	d.activeList = nodes

	return nodes
}

// Estimate the number of member nodes that have not been marked as dead,
// excluding the local node.
func (d *Detector) ActiveCount() int {
	return int(atomic.LoadInt64(&d.activeCount))
}

// Run the failure detector loop.
func (d *Detector) loop() {
	var probedNodes []*InternalNode

	ticker := time.NewTicker(d.ProbeInterval)
	defer ticker.Stop()

	timer := time.NewTimer(0)
	timer.Stop()
	defer timer.Stop()

	for {
		select {
		case <-d.stopping: // stop signal
			d.stopped <- struct{}{}
			return

		case <-timer.C: // probe timeout
			d.l.Lock()
			if !d.period.IsZero() && probedNodes != nil {
				d.indirectProbe(probedNodes)
			}
			d.l.Unlock()

		case t := <-ticker.C: // protocol period
			d.l.Lock()

			// handle suspicion from the previous protocol period
			if !d.period.IsZero() && probedNodes != nil {
				d.suspected(probedNodes)
			}
			d.period = t

			// send out the probes
			probedNodes = d.probe()

			d.l.Unlock()

			// set the timer for maybe sending indirect probes
			timer.Reset(d.boundedTimeout(probedNodes))
		}
	}
}

// Send fresh probes.
func (d *Detector) probe() (nodes []*InternalNode) {

	// useful for testing
	if d.DirectProbes == 0 {
		return nil
	}

	// maximum number of probes
	max := d.nodes.Len()
	probes := int(d.DirectProbes)

	// send the probes
	for i, j := 0, 0; i < max && j < probes; i += 1 {
		node := d.nodes.Next()
		if node == nil {
			// no-op
		} else if node.State == Dead {
			d.stateUpdate(node, Dead, false)
		} else {
			if len(node.Addrs) > 0 {
				d.sendTo(node, d.ping())
				j += 1
			}
			// append to send indirect request
			nodes = append(nodes, node)
		}
	}

	return
}

// Send indirect probes.
func (d *Detector) indirectProbe(nodes []*InternalNode) {

	// so that we don't ask the probe targets to probe themselves
	flags := make(map[uint64]bool)

	// batch requests for the indirect probes
	requests := []interface{}{}
	for _, node := range nodes {
		if node.LastAckTime.IsZero() || node.LastAckTime.Before(d.period) ||
			len(node.Addrs) == 0 {
			requests = append(requests, d.pingRequest(node))
			flags[node.Id] = true
		}
	}

	// nothing to send
	if len(requests) == 0 {
		return
	}

	// also don't ask suspect nodes for indirect probes
	for id := range d.suspects {
		flags[id] = true
	}

	// consider up to the configured value or the number of active nodes
	max := d.ActiveCount() - len(flags)
	if int(d.IndirectProbes) < max {
		max = int(d.IndirectProbes)
	}

	// send the indirect probe requests
	for i := 0; i < max; {
		if node := d.nodes.Next(); node != nil && !flags[node.Id] {
			d.sendTo(node, requests...)
			i += 1
		}
	}
}

// Send suspect events.
func (d *Detector) suspected(nodes []*InternalNode) {

	// these nodes have not responded since the last protocol period
	for _, node := range nodes {
		if node.LastAckTime.IsZero() || node.LastAckTime.Before(d.period) {
			if node.State != Suspect {
				d.stateUpdate(node, Suspect, true)
			}
		}
	}

	// nodes that have not disputed their suspect status before this time are
	// considered dead
	deathTime := time.Now().Add(-d.SuspicionDuration())

	// determine which nodes have died
	for id, node := range d.suspects {

		// if the node is not suspect, then it's alive or dead (as a cat)
		if node.State != Suspect {
			delete(d.suspects, id)
			continue
		}

		// node is still suspect
		if node.SuspectTime.IsZero() || node.SuspectTime.After(deathTime) {
			continue
		}

		// the node is dead if it hasn't disputed its suspicion since it became
		// suspected of failure
		d.stateUpdate(node, Dead, true)
		delete(d.suspects, id)
	}
}

// Receive messages from the network.
func (d *Detector) recv() {

	// loop while we're active
	// the state prevents multiple concurrent goroutines
	for state := d.state; d.started && d.state == state; {

		// receive message from broker
		msg, err := d.broker.Recv()

		// drop this message if we're stopped
		if !d.started {
			return
		}

		// log errors
		if err != nil {
			if d.Logger != nil {
				d.Logger.Printf("[recv %v] %v", d.LocalNode.Id, err)
			}
			continue
		}

		// log the message
		if d.Logger != nil {
			d.Logger.Printf("[recv %v] %v", d.LocalNode.Id, msg)
		}

		// handle the message
		d.handle(msg)
	}
}

func (d *Detector) handle(msg *Message) {
	d.l.Lock()

	// witness global incarnation number
	d.incarnation.Witness(msg.Incarnation)

	// just in case, ignore messages from self
	if msg.From == d.LocalNode.Id {
		return
	}

	// anti-entropy
	events := msg.Events()
	if msg.To == d.LocalNode.Id {

		// process anti-entropy event
		if len(events) > 0 {
			if event, ok := events[0].(*AntiEntropyEvent); ok {
				d.handleEvent(event)
				events = events[1:]
			}
		}

		node := d.lookup(msg.From, nil)
		node.RemoteIncarnation = msg.Incarnation

		// maybe dispute local state
		if d.LocalNode.Incarnation.Compare(msg.Incarnation) < 0 {
			d.Broadcast(d.aliveNode(&d.LocalNode))
		}
	}

	// queue events
	for _, event := range events {
		d.handleEvent(event)
	}

	d.l.Unlock()

	// trigger message update
	if d.MessageCh != nil {
		d.MessageCh <- *msg
	}
}

func (d *Detector) handleEvent(event interface{}) {
	switch event := event.(type) {
	case PingEvent:
		d.handlePing(&event)

	case AckEvent:
		d.handleAck(&event)

	case IndirectPingRequestEvent:
		d.handleIndirectPingRequest(&event)

	case IndirectPingEvent:
		d.handleIndirectPing(&event)

	case IndirectAckEvent:
		d.handleIndirectAck(&event)

	case AntiEntropyEvent:
		d.handleAntiEntropy(&event)

	case AliveEvent:
		d.handleAlive(&event)

	case SuspectEvent:
		d.handleSuspect(&event)

	case DeathEvent:
		d.handleDeath(&event)

	case UserEvent:
		d.handleUserEvent(&event)

	default:
		if d.Logger != nil {
			d.Logger.Printf("[handle] Unrecognized event %v", event)
		}
	}
}

// Handle pings.
func (d *Detector) handlePing(event *PingEvent) {

	// lookup the node
	node := d.lookup(event.From, nil)

	// just in case, ignore pings from self
	if event.From == d.LocalNode.Id {
		return
	}

	// can't acknowledge without return address
	if len(node.Addrs) == 0 {
		return
	}

	// acknowledge the ping
	d.sendTo(node, d.ack(event.Time))
}

// Handle indirect ping requests.
func (d *Detector) handleIndirectPingRequest(event *IndirectPingRequestEvent) {

	// just in case, ignore pings from self or to self
	if event.From == d.LocalNode.Id || event.Target == d.LocalNode.Id {
		return
	}

	// lookup the nodes
	from := d.lookup(event.From, event.Addrs)
	target := d.lookup(event.Target, event.TargetAddrs)

	// special case for dead nodes
	if target.State == Dead {
		d.sendTo(from, d.death(target))
		return
	}

	// anti-entropy
	if len(event.TargetAddrs) == 0 {
		if len(target.Addrs) == 0 {
			return
		}
		d.sendTo(from, d.antiEntropy(&target.Node))
	}

	// send indirect ping
	d.sendTo(target, d.pingVia(from, event.Time))
}

// Handle indirect pings.
func (d *Detector) handleIndirectPing(event *IndirectPingEvent) {

	// just in case, ignore pings from self or to self
	if event.From == d.LocalNode.Id || event.Via == d.LocalNode.Id {
		return
	}

	// lookup the nodes
	node := d.lookup(event.From, event.Addrs)
	via := d.lookup(event.Via, event.ViaAddrs)

	// send indirect ack
	d.sendTo(node, d.indirectAck(event.Time, via, event.ViaTime))
}

// Handle acknowledgements.
func (d *Detector) handleAck(event *AckEvent) {

	// just in case, ignore acks from self
	if event.From == d.LocalNode.Id {
		return
	}

	// lookup the node
	node := d.lookup(event.From, nil)

	// check the timestamp
	if d.period.IsZero() || node.LastAckTime.IsZero() {
		// no-op
	} else if node.LastAckTime.After(d.period) {
		// node already acknowledged
		return
	} else if event.Time.IsZero() || event.Time.Before(d.period) {
		// ignore if invalid time or very late response
		return
	}

	// update RTT; this extends the RTT in the case of an indirect ack to
	// reduce the likelihood of future false negatives from slow nodes
	node.RTT.Update(time.Since(event.Time))

	// set last ack time
	node.LastAckTime = time.Now()

	// send alive message if node isn't marked as alive
	if node.State != Alive {
		d.stateUpdate(node, Alive, true)
	}
}

// Handle indirect acknowledgement.
func (d *Detector) handleIndirectAck(event *IndirectAckEvent) {

	// just in case, ignore acks from self
	if event.Via == d.LocalNode.Id {
		return
	}

	// handle the ack locally
	d.handleAck(&event.AckEvent)

	// lookup the node
	node := d.lookup(event.Via, nil)

	// update ack event for requesting node
	ack := &event.AckEvent
	ack.Time = event.ViaTime

	// relay ack to requestor
	d.sendTo(node, ack)
}

// Handle anti-entropy event.
func (d *Detector) handleAntiEntropy(event *AntiEntropyEvent) {

	// just in case, ignore anti-entropy from self
	if event.Id == d.LocalNode.Id {
		d.Broadcast(d.aliveNode(&d.LocalNode))
		return
	}

	// lookup the node
	node := d.lookup(event.Id, nil)

	// witness global incarnation number
	d.incarnation.Witness(event.Incarnation)

	// ignore old updates
	if node.Incarnation.Compare(event.Incarnation) >= 0 {
		return
	}

	// update the node
	node.Node = event.Node

	// trigger state update
	d.stateUpdate(node, event.State, false)
}

// Handle alive event.
func (d *Detector) handleAlive(event *AliveEvent) {
	d.handleStateBroadcast(event, event.Id, event.Incarnation, Alive)
}

// Handle suspect event.
func (d *Detector) handleSuspect(event *SuspectEvent) {
	d.handleStateBroadcast(event, event.Id, event.Incarnation, Suspect)
}

// Handle death event.
func (d *Detector) handleDeath(event *DeathEvent) {
	d.handleStateBroadcast(event, event.Id, event.Incarnation, Dead)
}

// Handle the alive, suspect, and death state broadcasts.
func (d *Detector) handleStateBroadcast(event BroadcastEvent, id uint64, incarnation Seq, state State) {

	// witness global incarnation number
	d.incarnation.Witness(incarnation)

	// if self
	if id == d.LocalNode.Id {
		cmp := d.LocalNode.Incarnation.Compare(incarnation)
		// if our incarnation number is less than the state broadcast or
		// if our incarnation number is the same but the state isn't alive
		if cmp < 0 || state != Alive {
			// then we dispute the update
			d.LocalNode.Incarnation.Witness(d.incarnation.Increment())
			d.Broadcast(d.aliveNode(&d.LocalNode))
		}
		return
	}

	// lookup the node
	node := d.lookup(id, nil)

	if cmp := node.Incarnation.Compare(incarnation); cmp < 0 {

		// update incarnation numbers
		node.Incarnation.Witness(incarnation)

		// special case for alive
		if state == Alive {
			node.Node = event.(*AliveEvent).Node
		}

		// trigger state update for this new incarnation
		d.stateUpdate(node, state, false)

	} else if cmp > 0 {

		// we have an update to broadcast
		d.stateUpdate(node, state, false)

	}
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
		From: d.LocalNode.Id,
		Time: time.Now(),
	}
}

// Acknowledge a ping.
func (d *Detector) ack(t time.Time) *AckEvent {
	return &AckEvent{
		From: d.LocalNode.Id,
		Time: t,
	}
}

// Send an indirect ping request.
func (d *Detector) pingRequest(node *InternalNode) *IndirectPingRequestEvent {
	return &IndirectPingRequestEvent{
		From:        d.LocalNode.Id,
		Addrs:       d.LocalNode.Addrs,
		Time:        time.Now(),
		Target:      node.Id,
		TargetAddrs: node.Addrs,
	}
}

// Indirectly ping a node.
func (d *Detector) pingVia(via *InternalNode, viaTime time.Time) *IndirectPingEvent {
	return &IndirectPingEvent{
		PingEvent: *d.ping(),
		Addrs:     d.LocalNode.Addrs,
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
	return d.aliveNode(&node.Node)
}

// Broadcast news that a node is alive.
func (d *Detector) aliveNode(node *Node) *AliveEvent {
	return &AliveEvent{
		From: d.LocalNode.Id,
		Node: *node,
	}
}

// Broadcast news that a node is suspected of failure.
func (d *Detector) suspect(node *InternalNode) *SuspectEvent {
	return &SuspectEvent{
		From:        d.LocalNode.Id,
		Id:          node.Id,
		Incarnation: node.Incarnation.Get(),
	}
}

// Broadcast news that a node has died.
func (d *Detector) death(node *InternalNode) *DeathEvent {
	return d.deathNode(&node.Node)
}

// Broadcast news that a node has died.
func (d *Detector) deathNode(node *Node) *DeathEvent {
	return &DeathEvent{
		From:        d.LocalNode.Id,
		Id:          node.Id,
		Incarnation: node.Incarnation.Get(),
	}
}

// Send an anti-entropy event.
func (d *Detector) antiEntropy(node *Node) *AntiEntropyEvent {
	return &AntiEntropyEvent{
		Node: *node,
	}
}

// Send events to a node.
func (d *Detector) sendTo(node *InternalNode, events ...interface{}) {

	// can't send if there are no addresses
	if len(node.Addrs) == 0 {
		if d.Logger != nil {
			d.Logger.Printf("[send %v] Can't send to node %v with no addresses! %v", d.LocalNode.Id, node.Id, events)
		}
		return
	}

	// create the message
	msg := new(Message)
	msg.From = d.LocalNode.Id
	msg.To = node.Id
	msg.Incarnation = node.Incarnation

	// add anti-entropy first, if needed, to be processed first at remote node
	if i := d.LocalNode.Incarnation.Get(); node.RemoteIncarnation.Compare(i) < 0 {
		msg.AddEvent(d.antiEntropy(&d.LocalNode))
		node.RemoteIncarnation.Witness(i)
	}

	// add the event
	msg.AddEvent(events...)

	// re-broadcast self at low priority
	// d.broker.BroadcastL(d.aliveNode(&d.LocalNode))
	// for _, node := range d.nodeMap {
	// 	if node.State == Dead {
	// 		d.broker.BroadcastL(d.death(node))
	// 	}
	// }

	// send the message with piggybacked broadcasts
	d.broker.SetBroadcastLimit(d.RetransmitLimit())
	d.broker.SendTo(node.Addrs, msg)

	if d.Logger != nil {
		d.Logger.Printf("[send %v] %v", d.LocalNode.Id, msg)
	}
}

// Get the singleton node for the given ID.
func (d *Detector) lookup(id uint64, addrs []string) *InternalNode {

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

		// anti-entropy broadcasts
		for id := range d.actives {
			d.stateBroadcast(d.nodeMap[id])
		}

		// save node
		d.nodeMap[id] = node
	}

	// populate addresses
	if len(node.Addrs) == 0 {
		node.Addrs = addrs
	}

	// return the singleton
	return node
}

// Consolidate node state updates.
func (d *Detector) stateUpdate(node *InternalNode, state State, reincarnate bool) {

	// special handling
	switch state {

	case Suspect:

		// save into global suspect list
		d.suspects[node.Id] = node
		if node.State != Suspect {
			node.SuspectTime = d.period
		}
		fallthrough

	case Alive:

		// add to selection list
		if !d.actives[node.Id] {
			d.nodes.Add(node)
			d.actives[node.Id] = true
			d.activeList = nil
		}

	case Dead:

		// remove from selection list
		if d.actives[node.Id] {
			d.nodes.Remove(node)
			delete(d.actives, node.Id)
			d.activeList = nil
		}

	default: // unknown state
		return

	}

	// update node state
	node.State = state

	// remove from suspects list
	if state != Suspect {
		delete(d.suspects, node.Id)
	}

	// update active count
	atomic.StoreInt64(&d.activeCount, int64(len(d.actives)))

	// reincarnate
	if reincarnate {
		node.Incarnation.Witness(d.incarnation.Increment())
	}

	// broadcast change in state
	d.stateBroadcast(node)

	// notify update
	if d.UpdateCh != nil {
		d.l.Unlock()
		d.UpdateCh <- node.Node
		d.l.Lock()
	}
}

// Broadcast the current state of the given node.
func (d *Detector) stateBroadcast(node *InternalNode) {
	switch node.State {
	case Alive:
		d.Broadcast(d.alive(node))
	case Suspect:
		d.Broadcast(d.suspect(node))
	case Dead:
		d.Broadcast(d.death(node))
	}
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

// Calculate the suspicion duration after which a node is considered dead.
func (d *Detector) SuspicionDuration() time.Duration {
	// the suspicion time is calculated as mult*log(N+1); division by three is
	// to convert from log base 2 to base 10 (approximately)
	n := d.ActiveCount()
	i := log2ceil(int(n)+1) / 3
	if i < 1 {
		i = 1
	}
	return time.Duration(d.SuspicionMult) * time.Duration(i) * d.ProbeInterval
}

// Calculate the retransmission limit for broadcasts.
func (d *Detector) RetransmitLimit() uint {
	// calculate the retransmission limit as mult*log(N+1); the division by three
	n := d.ActiveCount()
	i := log2ceil(int(n)+1) / 3
	if i < 1 {
		i = 1
	}
	return d.RetransmitMult * uint(i)
}
