package swim

import (
	"testing"
)

func TestBroadcastQueue(t *testing.T) {
	bqueue := NewBroadcastQueue()

	// peek on empty queue
	if bqueue.Len() != 0 {
		t.Fatalf("Expected length of 0")
	}

	prune := func(b *Broadcast) func(b *Broadcast) bool {
		return func(that *Broadcast) bool {
			return b == nil || b == that
		}
	}

	// class 0 broadcast
	done := make(chan struct{}, 1)
	event0 := &SuspectEvent{From: 1, Id: 2, Incarnation: Seq(3)}
	bcast0 := &Broadcast{Class: 0, Event: event0, Done: done}
	bqueue.Push(bcast0)
	if bqueue.Len() != 1 {
		t.Fatalf("Expected length of 1")
	} else if bqueue.List()[0] != bcast0 {
		t.Fatalf("Expected list of broadcast 0")
	}

	bqueue.Prune(prune(bcast0))
	if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected length of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// class 1 broadcast
	event1 := &SuspectEvent{From: 2, Id: 3, Incarnation: Seq(4)}
	bcast1 := &Broadcast{Class: 1, Event: event1}
	bqueue.Push(bcast1)
	if bqueue.List()[0] != bcast1 {
		t.Fatalf("Expected list of broadcast 1")
	} else if bqueue.Len() != 1 {
		t.Fatalf("Expected length of 1")
	}

	bqueue.Prune(prune(bcast1))
	if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected length of 0 got %v", l)
	}

	// class 0 and 1 broadcasts w/ stable sort
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	if bqueue.Len() != 2 {
		t.Fatalf("Expected length of 2")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1 got %v", bqueue.List())
	}

	bqueue.Prune(prune(bcast1))
	if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected length of 1 got %v", l)
	} else if bqueue.List()[0] != bcast0 {
		t.Fatalf("Expected list of broadcast 0")
	}

	bqueue.Prune(prune(bcast0))
	if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected length of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// class 0 and 1 broadcasts w/ attempts after insertion
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	if bqueue.Len() != 2 {
		t.Fatalf("Expected length of 2")
	} else if bs := bqueue.List(); bs[0] != bcast0 || bs[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1 got %v and %v", bs[0], bs[1])
	}

	bqueue = NewBroadcastQueue()

	// class 0 and 1 broadcasts w/ attempts before insertion
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	if bqueue.Len() != 2 {
		t.Fatalf("Expected length of 2")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1 got %v", bqueue.List())
	}

	bqueue = NewBroadcastQueue()

	// invalidation
	event1p := &SuspectEvent{From: 2, Id: 3, Incarnation: Seq(5)}
	bcast1p := &Broadcast{Class: 1, Event: event1p}
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	bqueue.Push(bcast1p)
	if bqueue.Len() != 2 {
		t.Fatalf("Expected length of 2 got %v", bqueue.Len())
	} else if bs := bqueue.List(); bs[0] != bcast0 || bs[1] != bcast1p {
		t.Fatalf("Expected list of broadcast 0 and 1")
	}

	bqueue = NewBroadcastQueue()

	// three events
	bcast1.Attempts = 11
	event2 := &SuspectEvent{From: 3, Id: 4, Incarnation: Seq(5)}
	bcast2 := &Broadcast{Class: 1, Event: event2, Attempts: 10}
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	bqueue.Push(bcast2)
	if bqueue.Len() != 3 {
		t.Fatalf("Expected length of 3")
	} else if bs := bqueue.List(); bs[0] != bcast0 || bs[1] != bcast2 || bs[2] != bcast1 {
		t.Logf("%v %v %v", bs[0], bs[1], bs[2])
		t.Fatalf("Expected list of broadcast 0, 2, 1")
	}

	bqueue = NewBroadcastQueue()

	// test prune
	bqueue.Push(bcast1)
	bqueue.Push(bcast0)
	bqueue.Push(bcast2)
	bqueue.Prune(func(b *Broadcast) bool {
		return b == bcast0
	})
	if l := bqueue.Len(); l != 2 {
		t.Fatalf("Expected length of 2 got %v", l)
	}
}
