package main

import (
	"flag"
	"log"
	"math"
	"os"
	// "os/signal"
	"runtime"
	// "syscall"
	"time"

	. "github.com/mikepb/go-swim"
)

var verbose *bool = flag.Bool("verbose", false, "verbose")
var N *uint = flag.Uint("n", 16, "number of nodes")
var R *uint = flag.Uint("r", 8, "number of runs")
var K *uint = flag.Uint("k", 1, "number of buckets")
var P *uint = flag.Uint("p", 1, "number of direct probes")
var D *string = flag.String("d", "ring", "distance D")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// c := make(chan os.Signal, 1)
	// signal.Notify(c, syscall.SIGINFO)
	// go func() {
	// 	for range c {
	// 		buf := make([]byte, 8096*1024)
	// 		n := runtime.Stack(buf, true)
	// 		os.Stderr.Write(buf[:n])
	// 	}
	// }()

	flag.Parse()

	l := log.New(os.Stdout, "", 0)

	r := NewSimConvergenceRunner()
	r.K = *K
	r.P = *P

	d := *D
	switch d {
	case "xor":
		r.D = XorSorter
	case "finger":
		r.D = FingerSorter
	case "ring":
		r.D = RingSorter
	default:
		r.K = 1
		d = "none"
	}

	if *verbose {
		r.Logger = log.New(os.Stderr, "", 0)
	}

	// ts := make([]time.Duration, *R)
	// fs := make([]time.Duration, *R)
	for i := uint(0); i < *R; i += 1 {
		first, last := r.Measure(*N)
		l.Printf("%d\t%v\t%v\t%d\t%d\t%s", *N, first, last, r.K, r.P, d)
	}

	// fmean, fstddev := stat(fs)
	// tmean, tstddev := stat(ts)
	// l.Printf("%d\t%v\t%v\t%v\t%v\t%d\t%d\t%s\t%d", *N, fmean, fstddev, tmean, tstddev, r.K, r.P, d, *R)
}

func stat(ts []time.Duration) (mean, stddev time.Duration) {
	sum := float64(0)
	ss := float64(0)

	for _, t := range ts {
		sum += float64(t)
		ss += float64(t) * float64(t)
	}

	n := float64(len(ts))
	var2 := float64(0)
	if n > 1 {
		var2 = (n*ss - sum*sum) / (n * (n - 1))
	}

	mean = time.Duration(sum / n)
	stddev = time.Duration(math.Sqrt(var2))
	return
}
