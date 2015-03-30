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

	node := func(router *SimRouter) *Detector {
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
			ProbeTimeout:   20 * time.Millisecond,
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

	n1 := node(router)
	n2 := node(router)

	// start the nodes
	start()
	defer close()

	// these joins should be ignored
	n1.Join(n1.LocalNode.Addrs[0])
	n2.Join(n2.LocalNode.Addrs[0])

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

	// N1 should eventually suspect N2
	n2.Stop()
RETRY_SUSPECT:
	if u := <-n1.UpdateCh; u.State != Suspect {
		if u.State == Alive {
			goto RETRY_SUSPECT
		}
		t.Fatalf("N1 did not suspect N2 %v", u)
	}

	// N1 should eventually consider N2 alive
	n2.DirectProbes = 1
	n2.Start()
RETRY_REJOIN:
	if u := <-n1.UpdateCh; u.State != Alive {
		if u.State == Dead {
			goto RETRY_REJOIN
		}
		t.Fatalf("N1 did not consider N2 alive %v", u)
	}

	// N1 should eventually consider N2 dead
	n2.Leave()
RETRY_LEAVE:
	if u := <-n1.UpdateCh; u.State != Dead {
		if u.State == Suspect {
			goto RETRY_LEAVE
		}
		t.Fatalf("N1 did not consider N2 dead %v", u)
	}

	close()

	router = nil
	router_1_2 := NewSimRouter()
	router_2_3 := NewSimRouter()
	nodes = nil

	inode1 := node(router_2_3)
	inode2 := node(router_2_3)
	inode3 := node(router_2_3)

	// node 1 can only reach node 2
	router_1_2.Routes[inode2.LocalNode.Addrs[0]] = inode2.Transport.(*SimTransport)

	inode2.DirectProbes = 0
	inode3.DirectProbes = 0
	inode2.UpdateCh = nil
	inode3.UpdateCh = nil

	inode1.Start()
	inode2.Join(inode1.LocalNode.Addrs[0])

	// N1 should receive the join intent
	if u := <-inode1.UpdateCh; !reflect.DeepEqual(u, inode2.LocalNode) {
		t.Fatalf("N1 did not receive N2 join message %v != %v", u, inode2.LocalNode)
	}

	inode3.Join(inode1.LocalNode.Addrs[0])

	// N1 should receive the join intent
	if u := <-inode1.UpdateCh; !reflect.DeepEqual(u, inode3.LocalNode) {
		t.Fatalf("N1 did not receive N3 join message %v != %v", u, inode3.LocalNode)
	}

	inode1.UpdateCh = nil
	time.Sleep(time.Second)

	// N1 should consider N3 alive via indirect ping
	if n := inode1.ActiveCount(); n != 2 {
		t.Fatalf("N1 should report two active nodes got %v", n)
	}
}
