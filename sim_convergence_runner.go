package swim

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Run a simulator that measures the time for all nodes to agree on the
// number of members after a failure occurs.
type SimConvergenceRunner struct {
	Logger    *log.Logger
	K         uint
	P         uint
	D         Sorter
	l         sync.Mutex
	c         sync.Cond
	startTime time.Time
	firstTime time.Time
	subject   *Detector
	router    *SimRouter
	rand      *rand.Rand

	instances map[uint64]*Detector
	starts    map[uint64]bool
	expect    uint32
}

func NewSimConvergenceRunner() *SimConvergenceRunner {
	r := &SimConvergenceRunner{
		K:    1,
		P:    1,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	r.c.L = &r.l
	r.Reset()
	return r
}

func (r *SimConvergenceRunner) populate(n uint) {
	if l := len(r.instances); l > int(n) {
		for id, d := range r.instances {
			if r.Logger != nil {
				r.Logger.Printf("P REMOVE %v", d.LocalNode.Id)
			}
			delete(r.instances, id)
			delete(r.starts, id)
			l -= 1
			if l == int(n) {
				return
			}
		}
	} else {
		for i := l; i < int(n); i += 1 {
			d := r.newDetector()
			if r.Logger != nil {
				r.Logger.Printf("P NEW %v", d.LocalNode.Id)
			}
			r.instances[d.LocalNode.Id] = d
		}
	}
}

func (r *SimConvergenceRunner) newDetector() *Detector {
	for {
		id := uint64(r.rand.Int63())
		if _, ok := r.instances[id]; ok {
			continue
		}

		addr := fmt.Sprintf("n%020d", id)

		d := &Detector{
			LocalNode: Node{
				Id:    id,
				Addrs: []string{addr},
			},
			DirectProbes:   r.P,
			IndirectProbes: 3,
			ProbeInterval:  1000 * time.Millisecond,
			ProbeTimeout:   300 * time.Millisecond,
			RetransmitMult: 4,
			SuspicionMult:  5,
			Transport:      r.router.NewTransport(addr),
			Codec:          &FlateCodec{new(GobCodec)},
		}

		d.Logger = r.Logger
		d.SelectionList = r.newSelectionList(&d.LocalNode)
		d.UpdateCh = r.watch(d)

		r.instances[id] = d

		return d
	}
}

func (r *SimConvergenceRunner) newSelectionList(node *Node) SelectionList {
	if r.K <= 1 {
		return new(ShuffleList)
	} else {
		return &BucketList{
			K:         r.K,
			Sort:      r.D,
			LocalNode: node,
		}
	}
}

func (r *SimConvergenceRunner) watch(d *Detector) chan Node {
	ch := make(chan Node, 1)
	id := d.LocalNode.Id

	go func() {
		for {
			node, ok := <-ch

			// always broadcast to unblock
			r.c.Broadcast()

			if !ok || node.Id == 0 {
				break
			}

			// record first death
			r.l.Lock()
			if r.subject != nil && d != r.subject && r.firstTime.IsZero() &&
				node.Id == r.subject.LocalNode.Id && node.State == Dead {
				r.firstTime = time.Now()
			}
			r.l.Unlock()
		}

		r.l.Lock()
		delete(r.instances, id)
		delete(r.starts, id)
		delete(r.router.Routes, d.LocalNode.Addrs[0])
		r.c.Broadcast()
		r.l.Unlock()

		if r.Logger != nil {
			r.Logger.Printf("W EXIT %d", id)
		}
	}()

	return ch
}

func (r *SimConvergenceRunner) isDone() bool {
	expect := int(atomic.LoadUint32(&r.expect))
	for id, d := range r.instances {
		if !r.starts[id] {
			continue
		}
		c := d.ActiveCount()
		if r.Logger != nil {
			r.Logger.Printf("W COMPARE %d <> %d", c, expect)
		}
		if c != expect {
			return false
		}
	}
	return true
}

func (r *SimConvergenceRunner) Measure(n uint) (first, last time.Duration) {
	runtime.GC()

	r.l.Lock()
	defer r.l.Unlock()

	if r.Logger != nil {
		r.Logger.Println("M POPULATE")
	}
	r.populate(n)

	if r.Logger != nil {
		r.Logger.Println("M START")
	}
	atomic.StoreUint32(&r.expect, uint32(n)-1)
	r.start()

	for !r.isDone() {
		if r.Logger != nil {
			r.Logger.Println("M WAIT START")
		}
		r.c.Wait()
	}

	// ensure we reach steady state
	for _, d := range r.instances {
		time.Sleep(d.SuspicionDuration())
		break
	}

	if r.Logger != nil {
		r.Logger.Println("M KILL")
	}
	atomic.StoreUint32(&r.expect, uint32(n)-2)
	r.kill()
	t := time.Now()
	r.firstTime = time.Time{}

	for !r.isDone() {
		if r.Logger != nil {
			r.Logger.Println("M WAIT KILL")
		}
		r.c.Wait()
	}
	now := time.Now()

	if !r.firstTime.IsZero() {
		first = r.firstTime.Sub(t)
	} else {
		first = 365 * 24 * time.Hour
	}
	last = now.Sub(t)

	if r.Logger != nil {
		r.Logger.Println("M DONE")
	}
	return
}

func (r *SimConvergenceRunner) start() {

	// collect all addresses
	addrs := []string(nil)
	for _, d := range r.instances {
		addrs = append(addrs, d.LocalNode.Addrs...)
	}

	// start detectors
	for id, d := range r.instances {
		if r.starts[id] {
			continue
		}
		r.starts[id] = true
		if r.Logger != nil {
			r.Logger.Printf("S START %v", id)
		}

		d.Join(addrs...)
		r.l.Unlock()
		time.Sleep(time.Duration(r.rand.Int63n(int64(d.ProbeInterval))))
		r.l.Lock()
	}
}

func (r *SimConvergenceRunner) kill() {
	for id, d := range r.instances {
		if r.Logger != nil {
			r.Logger.Printf("K CLOSE %v", id)
		}
		r.subject = d
		// d.Stop()
		d.Close()
		d.UpdateCh <- Node{}
		delete(r.instances, id)
		delete(r.starts, id)
		if r.Logger != nil {
			r.Logger.Printf("K KILLED %v", id)
		}
		return
	}
}

func (r *SimConvergenceRunner) Reset() {
	r.l.Lock()
	defer r.l.Unlock()
	for _, d := range r.instances {
		d.Close()
		d.UpdateCh <- Node{}
	}
	r.subject = nil
	r.router = NewSimRouter()
	r.instances = make(map[uint64]*Detector)
	r.starts = make(map[uint64]bool)
}
