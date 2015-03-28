package main

import (
	// "flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	. ".."
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	router := NewSimRouter()
	nodes := []*Detector{}

	watch := func(name string, d *Detector) chan Node {
		ch := make(chan Node, 1)
		l := log.New(os.Stdout, name+",", log.Lmicroseconds)
		go func() {
			for {
				event := <-ch
				l.Printf(",%d,%d,%v,%v", d.ActiveCount(), d.RetransmitLimit(), d.SuspicionTime(), event)
			}
		}()
		return ch
	}

	nextId := 0
	node := func() *Detector {
		id := uint64(rand.Int63())
		name := fmt.Sprintf("n%04d", nextId)
		nextId += 1
		d := &Detector{
			LocalNode: Node{
				Id:    id,
				Addrs: []string{name},
			},
			DirectProbes:   1,
			IndirectProbes: 3,
			ProbeInterval:  200 * time.Millisecond,
			ProbeTimeout:   10 * time.Millisecond,
			RetransmitMult: 4,
			SuspicionMult:  3,
			Transport:      router.NewTransport(name),
			Codec:          &LZ4Codec{new(GobCodec)},
			SelectionList:  new(ShuffleList),
			Logger:         log.New(os.Stderr, "", 0),
			// MessageCh:      make(chan Message),
		}
		d.UpdateCh = watch(name, d)
		nodes = append(nodes, d)
		return d
	}

	start := func() {
		for _, node := range nodes {
			node.Join(nodes[0].LocalNode.Addrs[0])
		}
	}

	close := func() {
		for _, node := range nodes {
			node.Close()
		}
	}

	for i := 0; i < 10; i += 1 {
		node()
	}

	start()
	defer close()

	time.Sleep(30 * time.Second)
}
