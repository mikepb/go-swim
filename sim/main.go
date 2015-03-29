package main

import (
	"flag"
	"log"
	"math"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	. "github.com/mikepb/go-swim"
)

var verbose *bool = flag.Bool("verbose", false, "verbose")
var N *uint = flag.Uint("n", 16, "number of nodes")
var R *uint = flag.Uint("r", 8, "number of runs")
var K *uint = flag.Uint("k", 1, "number of buckets")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINFO)
	go func() {
		for range c {
			buf := make([]byte, 8096*1024)
			n := runtime.Stack(buf, true)
			os.Stderr.Write(buf[:n])
		}
	}()

	flag.Parse()

	l := log.New(os.Stdout, "", 0)
	l.Printf("mean\tstddev\tn\tk\truns")

	for k := uint(1); k <= *K; k += 1 {
		r := NewSimConvergenceRunner()
		r.K = k
		if *verbose {
			r.Logger = log.New(os.Stderr, "", 0)
		}

		for n := uint(2); n <= *N; n += 1 {
			ts := make([]time.Duration, *R)

			for j := range ts {
				ts[j] = r.Measure(n)
				r.Reset()
			}

			mean, stddev := stat(ts)
			l.Printf("%v\t%v\t%d\t%d\t%d", mean, stddev, n, k, *R)
		}
	}
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
