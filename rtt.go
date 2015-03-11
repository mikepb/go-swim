package swim

import (
	"math"
	"sync/atomic"
	"time"
)

// RTT implements the Jacobson/Karels algorithm for estimating round-trip
// time delays. The formulation used is from Section 3.5 of Computer
// Networking: A Top-Down Approach by Kurose and Ross.
type RTT struct {
	mean int64
	dev  int64
}

// Set the mean RTT time to the given value and set the deviation to zero.
// Use this method to reset the RTT estimate after a known change in the
// network delay.
func (r *RTT) Hint(mean time.Duration) {
	atomic.StoreInt64(&r.mean, int64(mean))
	atomic.StoreInt64(&r.dev, 0)
}

// Atomically read the estimated mean RTT.
func (r *RTT) Mean() time.Duration {
	return time.Duration(atomic.LoadInt64(&r.mean))
}

// Atomically read the estimated RTT deviation.
func (r *RTT) Deviation() time.Duration {
	return time.Duration(atomic.LoadInt64(&r.dev))
}

// Update the RTT estimate using a RTT sample.
func (r *RTT) Sample(v time.Duration) {
	r.SampleWith(v, 0.125, 0.25)
}

// Update the RTT estimate using a RTT sample with the specified parameters.
func (r *RTT) SampleWith(v time.Duration, a, b float64) {
	mean := atomic.LoadInt64(&r.mean)
	dev := atomic.LoadInt64(&r.dev)

	// calculate new estimates
	newMean := int64((1.0-a)*float64(mean) + a*float64(v))
	newDev := int64((1.0-b)*float64(dev) + b*math.Abs(float64(v)-float64(mean)))

	// atomically update estimates
	atomic.CompareAndSwapInt64(&r.mean, mean, newMean)
	atomic.CompareAndSwapInt64(&r.dev, dev, newDev)
}

// Calculate the estimated RTT.
func (r *RTT) Timeout() time.Duration {
	return r.Mean() + 4*r.Deviation()
}
