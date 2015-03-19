package swim

import (
	"testing"
)

func TestBroadcastQueue(t *testing.T) {
	bqueue := NewBroadcastQueue()

	// peek on empty queue
	if bqueue.Peek() != nil {
		t.Fatalf("Expected peek to return nil")
	} else if bqueue.Len() != 0 {
		t.Fatalf("Expected lenght of 0")
	}

	// class 0 broadcast
	done := make(chan struct{}, 1)
	event0 := &SuspectEvent{From: 1, Id: 2, Incarnation: Seq(3)}
	bcast0 := &Broadcast{Class: 0, Event: event0, Done: done}
	bqueue.Push(bcast0)
	if bqueue.Peek() != bcast0 {
		t.Fatalf("Expected peek of broadcast 0")
	} else if bqueue.Len() != 1 {
		t.Fatalf("Expected lenght of 1")
	} else if bqueue.List()[0] != bcast0 {
		t.Fatalf("Expected list of broadcast 0")
	} else if bqueue.Pop() != bcast0 {
		t.Fatalf("Expected pop of broadcast 0")
	} else if l := bqueue.Len(); l != 0 {
		t.Log(bqueue)
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// class 1 broadcast
	event1 := &SuspectEvent{From: 2, Id: 3, Incarnation: Seq(4)}
	bcast1 := &Broadcast{Class: 1, Event: event1}
	bqueue.Push(bcast1)
	if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if bqueue.List()[0] != bcast1 {
		t.Fatalf("Expected list of broadcast 1")
	} else if bqueue.Len() != 1 {
		t.Fatalf("Expected lenght of 1")
	} else if bqueue.Pop() != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// class 0 and 1 broadcasts w/ stable sort
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	if bqueue.Len() != 2 {
		t.Fatalf("Expected lenght of 2")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1")
	} else if bqueue.Peek() != bcast0 {
		t.Fatalf("Expected peek of broadcast 0")
	} else if bqueue.Pop() != bcast0 {
		t.Fatalf("Expected pop of broadcast 0")
	} else if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected lenght of 1 got %v", l)
	} else if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if bqueue.Pop() != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// class 0 and 1 broadcasts w/ attempts after insertion
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	bcast0.Attempts += 1
	bcast1.Attempts += 1
	bqueue.Init()
	if bqueue.Len() != 2 {
		t.Fatalf("Expected lenght of 2")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1")
	} else if bqueue.Peek() != bcast0 {
		t.Fatalf("Expected peek of broadcast 0")
	} else if bqueue.Pop() != bcast0 {
		t.Fatalf("Expected pop of broadcast 0")
	} else if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected lenght of 1 got %v", l)
	} else if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if bqueue.Pop() != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// class 0 and 1 broadcasts w/ attempts before insertion
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	if bqueue.Len() != 2 {
		t.Fatalf("Expected lenght of 2")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1")
	} else if bqueue.Peek() != bcast0 {
		t.Fatalf("Expected peek of broadcast 0")
	} else if bqueue.Pop() != bcast0 {
		t.Fatalf("Expected pop of broadcast 0")
	} else if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected lenght of 1 got %v", l)
	} else if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if bqueue.Pop() != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// invalidation
	event1p := &SuspectEvent{From: 2, Id: 3, Incarnation: Seq(5)}
	bcast1p := &Broadcast{Class: 1, Event: event1p}
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	bqueue.Push(bcast1p)
	bqueue.Init()
	if bqueue.Len() != 2 {
		t.Fatalf("Expected lenght of 2")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast1 {
		t.Fatalf("Expected list of broadcast 0 and 1")
	} else if bqueue.Peek() != bcast0 {
		t.Fatalf("Expected peek of broadcast 0")
	} else if bqueue.Pop() != bcast0 {
		t.Fatalf("Expected pop of broadcast 0")
	} else if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected lenght of 1 got %v", l)
	} else if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if p := bqueue.Pop(); p != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if p.Event != event1p {
		t.Fatalf("Expected broadcast 1 with second event")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// three events
	bcast1.Attempts = 11
	event2 := &SuspectEvent{From: 3, Id: 4, Incarnation: Seq(5)}
	bcast2 := &Broadcast{Class: 1, Event: event2, Attempts: 10}
	bqueue.Push(bcast0)
	bqueue.Push(bcast1)
	bqueue.Push(bcast2)
	bqueue.Init()
	if bqueue.Len() != 3 {
		t.Fatalf("Expected lenght of 3")
	} else if bqueue.List()[0] != bcast0 || bqueue.List()[1] != bcast2 || bqueue.List()[2] != bcast1 {
		t.Fatalf("Expected list of broadcast 0, 2, 1")
	} else if bqueue.Peek() != bcast0 {
		t.Fatalf("Expected peek of broadcast 0")
	} else if bqueue.Pop() != bcast0 {
		t.Fatalf("Expected pop of broadcast 0")
	} else if l := bqueue.Len(); l != 2 {
		t.Fatalf("Expected lenght of 2 got %v", l)
	} else if bqueue.Peek() != bcast2 {
		t.Fatalf("Expected peek of broadcast 2")
	} else if bqueue.Pop() != bcast2 {
		t.Fatalf("Expected pop of broadcast 2")
	} else if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected lenght of 1 got %v", l)
	} else if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if bqueue.Pop() != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done

	// test prune
	bqueue.Push(bcast1)
	bqueue.Push(bcast0)
	bqueue.Push(bcast2)
	bqueue.Prune(func(b *Broadcast) bool {
		return b == bcast0
	})
	if l := bqueue.Len(); l != 2 {
		t.Fatalf("Expected lenght of 2 got %v", l)
	} else if bqueue.Peek() != bcast2 {
		t.Fatalf("Expected peek of broadcast 2")
	} else if bqueue.Pop() != bcast2 {
		t.Fatalf("Expected pop of broadcast 2")
	} else if l := bqueue.Len(); l != 1 {
		t.Fatalf("Expected lenght of 1 got %v", l)
	} else if bqueue.Peek() != bcast1 {
		t.Fatalf("Expected peek of broadcast 1")
	} else if bqueue.Pop() != bcast1 {
		t.Fatalf("Expected pop of broadcast 1")
	} else if l := bqueue.Len(); l != 0 {
		t.Fatalf("Expected lenght of 0 got %v", l)
	}

	// go test will detect deadlock if we were not signalled
	<-done
}
