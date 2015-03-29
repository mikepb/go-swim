package swim

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var nilRunnerLogger = log.New(ioutil.Discard, "[runner] ", 0)

// Run a simulator that measures the time for all nodes to agree on the
// number of members after a failure occurs.
type SimConvergenceRunner struct {
	Logger    *log.Logger
	K         uint
	l         sync.Mutex
	c         sync.Cond
	startTime time.Time
	stepTime  time.Time
	router    *SimRouter
	rand      *rand.Rand

	instances map[uint64]*Detector
	starts    map[uint64]bool
	expect    uint32
	done      bool
}

func NewSimConvergenceRunner() *SimConvergenceRunner {
	r := &SimConvergenceRunner{
		Logger: nilRunnerLogger,
		K:      1,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	r.c.L = &r.l
	r.Reset()
	return r
}

func (r *SimConvergenceRunner) populate(n uint) {
	if l := len(r.instances); l > int(n) {
		for id, d := range r.instances {
			r.Logger.Printf("P REMOVE %v", d.LocalNode.Id)
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
			r.Logger.Printf("P NEW %v", d.LocalNode.Id)
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
			DirectProbes:   1,
			IndirectProbes: 3,
			ProbeInterval:  200 * time.Millisecond,
			ProbeTimeout:   20 * time.Millisecond,
			RetransmitMult: 4,
			SuspicionMult:  3,
			Transport:      r.router.NewTransport(addr),
			Codec:          &LZ4Codec{new(GobCodec)},
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
			Sort:      RingSorter,
			LocalNode: node,
		}
	}
}

func (r *SimConvergenceRunner) watch(d *Detector) chan Node {
	ch := make(chan Node, 1)
	id := d.LocalNode.Id

	go func() {
		for {
			if node, ok := <-ch; !ok || node.Id == 0 {
				break
			}
			if r.isDone() {
				r.done = true
				r.Logger.Printf("W DONE")
				r.c.Broadcast()
			}
		}

		defer r.Logger.Printf("W EXIT %d", id)
		r.l.Lock()
		delete(r.instances, id)
		delete(r.starts, id)
		delete(r.router.Routes, d.LocalNode.Addrs[0])
		r.c.Broadcast()
		r.l.Unlock()
	}()

	return ch
}

func (r *SimConvergenceRunner) isDone() bool {
	expect := int(atomic.LoadUint32(&r.expect))
	for _, d := range r.instances {
		c := d.ActiveCount()
		r.Logger.Printf("W COMPARE %d <> %d", c, expect)
		if c != expect {
			return false
		}
	}
	return true
}

func (r *SimConvergenceRunner) Measure(n uint) time.Duration {
	r.l.Lock()
	defer r.l.Unlock()

	r.Logger.Println("M POPULATE")
	r.populate(n)

	r.Logger.Println("M START")
	atomic.StoreUint32(&r.expect, uint32(n)-1)
	r.done = false
	r.start()

	for !r.isDone() {
		r.Logger.Println("M WAIT START")
		r.c.Wait()
	}

	r.Logger.Println("M KILL")
	atomic.StoreUint32(&r.expect, uint32(n)-2)
	r.done = false
	t := time.Now()
	r.kill()

	for !r.isDone() {
		r.Logger.Println("M WAIT KILL")
		r.c.Wait()
	}

	defer r.Logger.Println("M DONE")
	return time.Since(t)
}

func (r *SimConvergenceRunner) start() {
	// n := len(r.instances)

	// r.router.NetDelay = 0
	// r.router.NetStdDev = 0

	// collect all addresses
	addrs := []string(nil)
	for _, d := range r.instances {
		addrs = append(addrs, d.LocalNode.Addrs...)
	}

	// for _, d := range r.instances {
	// 	d.ProbeInterval = time.Duration(n*2) * time.Millisecond
	// 	d.ProbeTimeout = 1 * time.Millisecond
	// }

	for id, d := range r.instances {
		if r.starts[id] {
			continue
		}
		r.starts[id] = true
		r.Logger.Printf("S START %v", id)

		d.Join(addrs...)
		// r.l.Unlock()
		// r.Logger.Println("S UNLOCK")
		// time.Sleep(time.Duration(n*n*4) * time.Millisecond)
		// r.Logger.Println("S LOCK")
		// r.l.Lock()
	}

	// for _, d := range r.instances {
	// 	d.ProbeInterval = 200 * time.Millisecond
	// 	d.ProbeTimeout = 10 * time.Millisecond
	// }

	// r.router.NetDelay = kNetDelay
	// r.router.NetStdDev = kNetStdDev
}

func (r *SimConvergenceRunner) kill() {
	for id, d := range r.instances {
		r.Logger.Printf("K CLOSE %v", id)
		d.Close()
		d.UpdateCh <- Node{}
		r.Logger.Printf("K KILLED %v", id)
		delete(r.instances, id)
		delete(r.starts, id)
		return
	}
}

func (r *SimConvergenceRunner) Reset() {
	r.l.Lock()
	defer r.l.Unlock()
	for _, d := range r.instances {
		d.Close()
		close(d.UpdateCh)
	}
	r.router = NewSimRouter()
	r.instances = make(map[uint64]*Detector)
	r.starts = make(map[uint64]bool)
}
