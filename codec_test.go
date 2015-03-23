package swim

import (
	"reflect"
	"testing"
	"time"
)

func testCodec(t *testing.T, codec Codec) {
	msg := new(Message)

	test := func() {
		encode := &CodedMessage{Message: *msg}
		if err := codec.Encode(encode); err != nil {
			t.Fatal(err)
		}

		decode := &CodedMessage{Bytes: encode.Bytes}
		if err := codec.Decode(decode); err != nil {
			t.Fatal(err)
		} else if events := decode.Message.Events(); len(events) != len(msg.Events()) {
			t.Fatalf("Expected %v events in message got %v", len(msg.Events()), len(events))
		}

		events := decode.Message.Events()
		fixtures := msg.Events()

		for i, event := range events {
			t.Log(event)
			t.Log(fixtures[i])
			if !reflect.DeepEqual(event, fixtures[i]) {
				t.Fatalf("Expected %v got %v", fixtures, events)
			}
		}

		if encode.Size != len(encode.Bytes) {
			t.Fatalf("Size does not match byte length")
		}
	}

	// no events
	test()

	// ping event
	msg.AddEvent(PingEvent{From: 12, Incarnation: Seq(9), Time: time.Now()})
	test()

	// ack event
	msg.AddEvent(AckEvent{From: 13, Incarnation: Seq(7), Time: time.Now()})
	test()

	// indirect ping request event
	msg.AddEvent(IndirectPingRequestEvent{
		From: 12, Addrs: []string{"12"}, Target: 13, Time: time.Now(),
	})
	test()

	// indirect ping event
	msg.AddEvent(IndirectPingEvent{
		PingEvent: PingEvent{From: 12, Incarnation: Seq(9), Time: time.Now()},
		Addrs:     []string{"12"}, Via: 13, ViaTime: time.Now(),
	})
	test()

	// indirect ack event
	msg.AddEvent(IndirectAckEvent{
		AckEvent: AckEvent{From: 13, Incarnation: Seq(7), Time: time.Now()},
		Via:      13, ViaTime: time.Now(),
	})
	test()

	// anti-entropy event
	msg.AddEvent(AntiEntropyEvent{
		From: 13, Node: Node{Id: 14, Incarnation: Seq(29), State: Alive},
	})
	test()

	// alive event
	msg.AddEvent(AliveEvent{
		From: 13, Node: Node{Id: 14, Incarnation: Seq(29)},
	})
	test()

	// suspect event
	msg.AddEvent(SuspectEvent{From: 13, Id: 14, Incarnation: Seq(29)})
	test()

	// death event
	msg.AddEvent(DeathEvent{From: 13, Id: 14, Incarnation: Seq(29)})
	test()

	// user event
	msg.AddEvent(UserEvent{From: 13, Incarnation: Seq(29), Data: 1999})
	test()
}
