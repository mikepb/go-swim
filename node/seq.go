package node

import (
	"sync/atomic"
)

const maxSeq uint32 = 4294967295
const halfSeq uint32 = maxSeq / 2

// Seq atomically manipulates a 32-bit sequence number within a window size
// of 2^31 ala TCP sequence numbers. The window size for the sequence
// numbers should never span more than 2^31.
type Seq uint32

// Atomically read the sequence number.
func (s *Seq) Get() Seq {
	return Seq(atomic.LoadUint32((*uint32)(s)))
}

// Atomically increment the sequence number, returning the new value.
func (s *Seq) Increment() Seq {
	return Seq(atomic.AddUint32((*uint32)(s), 1))
}

// Increment this sequence to at least the given sequence.
func (s *Seq) Witness(v Seq) Seq {
	that := uint32(v)
	for {
		this := atomic.LoadUint32((*uint32)(s))

		// update this sequence only if less than the new sequence
		if that < this {
			return Seq(this)
		}

		// attempt to atomically update the sequence
		if atomic.CompareAndSwapUint32((*uint32)(s), this, that) {
			return Seq(that)
		}

		// another process succeeded in updating the sequence, try again
	}
}

// Compare the sequence to another sequence, taking a window size of 2^31
// into account.
func (s *Seq) Compare(v Seq) int {
	this := atomic.LoadUint32((*uint32)(s))
	that := uint32(v)

	if this < that {
		if that-this > halfSeq {
			// [this ... ... ... that]
			return 1
		} else {
			// [this ... that ... ...]
			return -1
		}
	} else if this > that {
		if this-that > halfSeq {
			// [that ... ... ... this]
			return -1
		} else {
			// [that ... this ... ...]
			return 1
		}
	} else {
		return 0
	}
}
