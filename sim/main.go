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
	startTime := time.Now()

	codec := &LZ4Codec{new(GobCodec)}
	router := NewSimRouter()
	nodes := []*Detector{}
	names := make(map[uint64]string)

	watch := func(name string, d *Detector) chan Node {
		ch := make(chan Node, 1)
		l := log.New(os.Stdout, name+",", log.Lmicroseconds)
		go func() {
			for {
				<-ch
				// event := <-ch
				// ns := ""
				// for _, node := range d.Members() {
				// 	ns = ns + " " + names[node.Id]
				// }
				// l.Printf(",%d,%d,%v,%s,%v", d.ActiveCount(), d.RetransmitLimit(), d.SuspicionDuration(), ns, event)
				l.Printf(",%d,%v", d.ActiveCount(), time.Since(startTime))
			}
		}()
		return ch
	}

	watchMsg := func(name string, d *Detector) chan Message {
		return nil
		ch := make(chan Message)
		go func() {
			for {
				msg := <-ch
				coded := &CodedMessage{Message: msg}
				codec.Encode(coded)
				log.Printf("%v", coded.Size)
			}
		}()
		return ch
	}

	nextId := 1
	node := func() *Detector {
		for {
			id := uint64(rand.Int63())
			if _, ok := names[id]; ok {
				continue
			}
			// id = uint64(nextId)
			name := fmt.Sprintf("n%04d", nextId)
			names[id] = name
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
				Codec:          codec,
				SelectionList:  new(ShuffleList),
				Logger:         log.New(os.Stderr, "", 0),
			}
			d.UpdateCh = watch(name, d)
			d.MessageCh = watchMsg(name, d)
			nodes = append(nodes, d)
			return d
		}
	}

	start := func() {
		startTime = time.Now()
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

	time.Sleep(5 * time.Second)
}
