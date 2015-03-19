package swim

import (
	"testing"
)

func TestBroadcast(t *testing.T) {
	var bcast Broadcast
	attempts := uint(0)

	reset := func() {
		bcast = Broadcast{}
		attempts = 0
	}

	priority := func(expect uint) {
		if p := bcast.Priority(); p != expect {
			t.Fatalf("Expected priority %v got %v", expect, p)
		}
	}

	attempt := func() {
		bcast.Attempts += 1
		attempts += 1
		if c := bcast.Attempts; c != attempts {
			t.Fatalf("Expected %v attempts got %v", attempts, c)
		}
	}

	// test priority class 0
	reset()
	priority(0)
	attempt()
	priority(0)
	attempt()
	priority(0)

	// test priority class 1
	reset()
	bcast.Class = 1
	priority(0)
	attempt()
	priority(1)
	attempt()
	priority(2)

	// test priority class 2
	reset()
	bcast.Class = 2
	priority(0)
	attempt()
	priority(2)
	attempt()
	priority(4)

	// test priority class 100
	reset()
	bcast.Class = 100
	priority(0)
	attempt()
	priority(100)
	attempt()
	priority(200)

	// test alive
	alive := &AliveEvent{From: 1, Node: Node{Id: 2, Incarnation: Seq(3)}}
	bcast.Event = alive
	aliveTag := bcast.Event.Tag()
	if aliveTag.From != 2 || aliveTag.Id != 2 || aliveTag.Type != bcastAlive {
		t.Fatalf("Expected valid alive tag")
	} else if bcast.Event.Seq().Get() != Seq(3) {
		t.Fatalf("Expected valid seq")
	}

	// test suspect
	suspect := &SuspectEvent{From: 4, Id: 5, Incarnation: Seq(6)}
	bcast.Event = suspect
	suspectTag := bcast.Event.Tag()
	if suspectTag.From != 5 || suspectTag.Id != 5 || suspectTag.Type != bcastSuspect {
		t.Fatalf("Expected valid suspect tag")
	} else if bcast.Event.Seq().Get() != Seq(6) {
		t.Fatalf("Expected valid seq")
	}

	// test death
	death := &DeathEvent{From: 7, Id: 8, Incarnation: Seq(9)}
	bcast.Event = death
	deathTag := bcast.Event.Tag()
	if deathTag.From != 8 || deathTag.Id != 8 || deathTag.Type != bcastDeath {
		t.Fatalf("Expected valid death tag")
	} else if bcast.Event.Seq().Get() != Seq(9) {
		t.Fatalf("Expected valid seq")
	}

	// test user
	user := &UserEvent{From: 10, Sequence: Seq(11)}
	bcast.Event = user
	userTag := bcast.Event.Tag()
	if userTag.From != 10 || userTag.Id != 11 || userTag.Type != bcastUser {
		t.Fatalf("Expected valid user tag")
	} else if bcast.Event.Seq().Get() != Seq(11) {
		t.Fatalf("Expected valid seq")
	}

	// non-invalidation
	bcast.Event = suspect
	that := bcast
	that.Event = death
	if bcast.Invalidates(&that) || that.Invalidates(&bcast) {
		t.Fatalf("Unexpected invalidation")
	}

	// equality
	that.Event = &SuspectEvent{From: 4, Id: 5, Incarnation: Seq(6)}
	if bcast.Invalidates(&that) || that.Invalidates(&bcast) {
		t.Fatalf("Unexpected invalidation")
	}

	// invalidation
	that.Event = &SuspectEvent{From: 4, Id: 5, Incarnation: Seq(7)}
	if !that.Invalidates(&bcast) || bcast.Invalidates(&that) {
		t.Fatalf("That should invalidate this")
	}

	// non-invalidation
	that.Event = &SuspectEvent{From: 4, Id: 5, Incarnation: Seq(5)}
	if that.Invalidates(&bcast) || !bcast.Invalidates(&that) {
		t.Fatalf("That should invalidate this")
	}
}
