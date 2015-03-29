package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	. ".."
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var lock sync.Mutex

	startTime := time.Now()
	stepTime := time.Now()

	codec := &LZ4Codec{new(GobCodec)}
	router := NewSimRouter()
	// router.NetDelay = 0
	// router.NetStdDev = 0
	nodes := []*Detector{}
	names := make(map[uint64]string)

	counts := make(map[uint64]int)
	stat := func() (float64, float64) {
		sum := float64(0)
		ss := float64(0)
		for _, c := range counts {
			sum += float64(c)
			ss += float64(c) * float64(c)
		}
		n := float64(len(counts))
		mean := sum / n
		var2 := (n*ss - sum*sum) / (n * (n - 1))
		stddev := math.Sqrt(var2)
		return mean, stddev
	}

	watch := func(name string, d *Detector) chan Node {
		ch := make(chan Node, 1)
		l := log.New(os.Stdout, name, 0)
		go func() {
			for {
				// event := <-ch
				// ns := ""
				// for _, node := range d.Members() {
				// 	ns = ns + " " + names[node.Id]
				// }
				// l.Printf(",%d,%d,%v,%s,%v", d.ActiveCount(), d.RetransmitLimit(), d.SuspicionDuration(), ns, event)
				<-ch
				lock.Lock()
				t := time.Since(startTime)
				s := time.Since(stepTime)
				count := d.ActiveCount()
				counts[d.LocalNode.Id] = count
				mean, stddev := stat()
				lock.Unlock()
				l.Printf(",%d,%f,%f,%v,%v", count, mean, stddev, t, s)
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
				ProbeTimeout:   20 * time.Millisecond,
				RetransmitMult: 4,
				SuspicionMult:  3,
				Transport:      router.NewTransport(name),
				Codec:          codec,
				SelectionList:  new(ShuffleList),
			}
			// d.Logger = log.New(os.Stderr, "", 0)
			d.UpdateCh = watch(name, d)
			d.MessageCh = watchMsg(name, d)
			nodes = append(nodes, d)
			return d
		}
	}

	start := func() {
		lock.Lock()
		defer lock.Unlock()

		startTime = time.Now()
		stepTime = startTime

		if len(nodes) == 0 {
			panic("No nodes configured!")
		}
		n1 := nodes[0]
		addrs := n1.LocalNode.Addrs

		if len(addrs) == 0 {
			panic("No addresses configured!")
		}

		for _, node := range nodes {
			node.Join(addrs...)
			lock.Unlock()
			time.Sleep(500 * time.Millisecond)
			lock.Lock()
		}
	}

	close := func() {
		lock.Lock()
		defer lock.Unlock()
		for _, node := range nodes {
			node.Close()
		}
	}

	kill := func() {
		lock.Lock()
		defer lock.Unlock()
		if len(nodes) == 0 {
			panic("No nodes configured!")
		}
		node := nodes[0]
		node.Close()
		delete(counts, node.LocalNode.Id)
		nodes = nodes[1:]
	}

	n := 64

	for i := 0; i < n; i += 1 {
		node()
	}

	start()
	defer close()

	time.Sleep(time.Duration(n) * 250 * time.Millisecond)

	lock.Lock()
	stepTime = time.Now()
	lock.Unlock()

	kill()

	time.Sleep(time.Duration(n) * 250 * time.Millisecond)
}
