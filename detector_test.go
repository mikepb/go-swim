package swim

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDetector(t *testing.T) {

	router := NewSimRouter()
	nodes := []*Detector{}

	node := func() *Detector {
		id := uint64(rand.Int63())
		name := fmt.Sprintf("node %v", id)
		d := &Detector{
			LocalNode: Node{
				Id:    id,
				Addrs: []string{name},
			},
			DirectProbes:   1,
			IndirectProbes: 3,
			ProbeInterval:  200 * time.Millisecond,
			ProbeTimeout:   10 * time.Millisecond,
			RetransmitMult: 3,
			SuspicionMult:  3,
			Transport:      router.NewTransport(name),
			Codec:          &LZ4Codec{new(GobCodec)},
			SelectionList:  new(ShuffleList),
			Logger:         log.New(os.Stdout, "", 0),
			UpdateCh:       make(chan Node, 1),
			// MessageCh:      make(chan Message),
		}
		nodes = append(nodes, d)
		return d
	}

	start := func() {
		for _, node := range nodes {
			node.Start()
		}
	}

	close := func() {
		for _, node := range nodes {
			node.Close()
		}
	}

	n1 := node()
	n2 := node()

	// start the nodes
	start()
	defer close()

	// join
	n2.Join(n1.LocalNode.Addrs[0])

	// N1 should receive the join intent
	if u := <-n1.UpdateCh; !reflect.DeepEqual(u, n2.LocalNode) {
		t.Fatalf("N1 did not receive N2 join message %v != %v", u, n2.LocalNode)
	}

	// N2 should receive anti-entropy from N1
	if u := <-n2.UpdateCh; !reflect.DeepEqual(u, n1.LocalNode) {
		t.Fatalf("N2 did not receive N1 anti-entropy %v != %v", u, n2.LocalNode)
	}

	// don't block updates
	n2.UpdateCh = nil

	// N1 should suspect N2
	n2.Stop()
	if u := <-n1.UpdateCh; u.State != Suspect {
		t.Fatalf("N1 did not suspect N2")
	}

	// N1 should consider N2 dead
	if u := <-n1.UpdateCh; u.State != Dead {
		t.Fatalf("N1 did not consider N2 dead %v", u)
	}

	// N1 should consider N2 alive
	n2.Start()
	if u := <-n1.UpdateCh; u.State != Alive {
		t.Fatalf("N1 did not consider N2 alive %v", u)
	}

	// stop N2 from pinging
	n2.DirectProbes = 0

	// N1 should suspect N1
	n2.Stop()
	if u := <-n1.UpdateCh; u.State != Suspect {
		t.Fatalf("N1 did not suspect N2")
	}

	// N1 should consider N2 alive
	n2.Start()
	if u := <-n1.UpdateCh; u.State != Alive {
		t.Fatalf("N1 did not consider N2 not suspect %v", u)
	}

	// leave
	n2.DirectProbes = 1
	n2.Leave()
	if u := <-n1.UpdateCh; u.State != Dead {
		t.Fatalf("N1 did not consider N2 dead %v", u)
	}


	// indirect probe

	// suspect
	// alive

	// suspect
	// dead

	// leave
}
