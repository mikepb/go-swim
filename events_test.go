package swim

import (
	"testing"
)

func TestBroadcastEvent(t *testing.T) {

	isBroadcast := func(b BroadcastEvent, tag BroadcastTag) {
		if b.Seq().Get() != Seq(13) {
			t.Fatalf("Expected sequence 13")
		}
		if b.Tag() != tag {
			t.Fatalf("Expected tag %v got %v", tag, b.Tag())
		}
	}

	tag := BroadcastTag{From: 12, Id: 34, Type: bcastAlive}
	isBroadcast(&AliveEvent{12, Node{Id: 34}, 13}, tag)

	tag.Type = bcastSuspect
	isBroadcast(&SuspectEvent{12, 34, 13}, tag)

	tag.Type = bcastDeath
	isBroadcast(&DeathEvent{12, 34, 13}, tag)

	tag.Id = 13
	tag.Type = bcastUser
	isBroadcast(&UserEvent{12, 13, nil}, tag)
}
