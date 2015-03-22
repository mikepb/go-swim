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

	tag := BroadcastTag{Id: 34, State: Alive}
	isBroadcast(&AliveEvent{12, Node{Id: 34, Incarnation: 13}}, tag)

	tag.State = Suspect
	isBroadcast(&SuspectEvent{12, 34, 13}, tag)

	tag.State = Dead
	isBroadcast(&DeathEvent{12, 34, 13}, tag)

	tag.Id = 13
	tag.State = 0
	isBroadcast(&UserEvent{34, 13, nil}, tag)
}
