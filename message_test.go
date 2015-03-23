package swim

import (
	"testing"
	"time"
)

func TestMessage(t *testing.T) {
	msg := new(Message)

	msg.AddEvent(PingEvent{From: 1, Time: time.Now()})
	if events := msg.Events(); len(events) != 1 {
		t.Fatalf("Expected %v events got %v", 1, len(events))
	}

	msg.AddEvent(PingEvent{From: 2, Time: time.Now()})
	if events := msg.Events(); len(events) != 2 {
		t.Fatalf("Expected %v events got %v", 2, len(events))
	}
}
