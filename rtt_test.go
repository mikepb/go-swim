package swim

import (
	"testing"
	"time"
)

func TestRTT(t *testing.T) {
	var rtt RTT

	if v := rtt.Mean(); v != 0 {
		t.Fatalf("expected mean to be %v got %v", 0, v)
	} else if v := rtt.Deviation(); v != 0 {
		t.Fatalf("expected deviation to be %v got %v", 0, v)
	}

	rtt.Update(time.Second)
	if u, v := 125*time.Millisecond, rtt.Mean(); v != u {
		t.Fatalf("expected mean to be %v got %v", u, v)
	} else if u, v := 250*time.Millisecond, rtt.Deviation(); v != u {
		t.Fatalf("expected deviation to be %v got %v", u, v)
	}

	rtt.UpdateWith(time.Second, 0.1, 0.5)
	if u, v := 212500*time.Microsecond, rtt.Mean(); v != u {
		t.Fatalf("expected mean to be %v got %v", u, v)
	} else if u, v := 562500*time.Microsecond, rtt.Deviation(); v != u {
		t.Fatalf("expected deviation to be %v got %v", u, v)
	} else if u, v := 2462500*time.Microsecond, rtt.Get(); u != v {
		t.Fatalf("expected estimate to be %v got %v", u, v)
	}

	rtt.Hint(200 * time.Millisecond)
	if u, v := 200*time.Millisecond, rtt.Mean(); v != u {
		t.Fatalf("expected mean to be %v got %v", u, v)
	} else if u, v := time.Duration(0), rtt.Deviation(); v != u {
		t.Fatalf("expected deviation to be %v got %v", u, v)
	}

	for i := 0; i < 9999; i++ {
		rtt.Update(time.Second)
	}

	if s, u, v := time.Second-time.Microsecond, time.Second+time.Microsecond, rtt.Mean(); s > v || v > u {
		t.Fatalf("expected mean to be within [%v...%v] got %v", s, u, v)
	} else if v := rtt.Get(); s > v || v > u {
		t.Fatalf("expected estimate to be within [%v...%v] got %v", s, u, v)
	} else if u, v := 1*time.Microsecond, rtt.Deviation(); v > u {
		t.Fatalf("expected deviation to be less than %v got %v", u, v)
	}
}
