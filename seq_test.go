package swim

import (
	"testing"
)

func TestSeq(t *testing.T) {
	s := new(Seq)

	if s.Get() != 0 {
		t.Fatalf("expected Seq to initialize to 0")
	}

	if s.Increment() != 1 {
		t.Fatalf("expected Seq to increment to 1")
	}

	if s.Get() != 1 {
		t.Fatalf("expected Seq to be 1")
	}

	if s.Witness(41) != 41 || s.Get() != 41 {
		t.Fatalf("expected Witness to return 41")
	}

	if s.Witness(41) != 41 || s.Get() != 41 {
		t.Fatalf("expected Witness to return 41")
	}

	if s.Witness(420) != 420 || s.Get() != 420 {
		t.Fatalf("expected Witness to return 420")
	}

	if s.Witness(420+Seq(halfSeq)) != 420 || s.Get() != 420 {
		t.Fatalf("expected Witness to return 420 got %v", s.Get())
	}

	if c := s.Compare(Seq(0)); c != 1 {
		t.Fatalf("expected Compare to return 1 got %v", c)
	}

	if c := s.Compare(Seq(420)); c != 0 {
		t.Fatalf("expected Compare to return 0 got %v", c)
	}

	if c := s.Compare(Seq(421)); c != -1 {
		t.Fatalf("expected Compare to return -1 got %v", c)
	}

	if c := s.Compare(Seq(halfSeq + 419)); c != -1 {
		t.Fatalf("expected Compare to return 1 got %v", c)
	}

	if c := s.Compare(Seq(halfSeq + 420)); c != 1 {
		t.Fatalf("expected Compare to return 1 got %v", c)
	}
}
