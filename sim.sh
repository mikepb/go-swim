#!/usr/bin/env sh

kmax=8
pmax=2
nmax=128
runs=8

# GOOS=linux go build -o simulate sim/main.go

trap 'exit' SIGHUP SIGINT SIGTERM

simulate() {
	local isdone=false
	while ! $isdone; do
		docker run -v $PWD:/sim -w /sim ubuntu:14.04 timeout -s 9 -k 20m 10m ./simulate -r 1 -n $1 -k $2 -p $3 -d $4
		if [ $? -eq 0 ]; then
			isdone=true
		fi
	done
}

# loop k kmax p pmax n nmax runs metric
loop() {
	local k=$1
	while [ $k -le $2 ]; do
		for p in `seq $3 $4`; do
			local n=$5
			while [ $n -le $6 ]; do
				for i in `seq 1 $7`; do
					simulate $n $k $p $8
				done
				n=`expr $n + $n`
			done
		done
		k=`expr $k + $k`
	done
}

echo "n	detection delay	broadcast delay	# buckets	# direct pings	metric"

# k=1 kmax=1 p=1 pmax=$pmax n=4 nmax=$nmax
loop 1 1 1 $pmax 4 $nmax $runs none

# k=1 kmax=$kmax p=2 pmax=$pmax n=4 nmax=$nmax
loop 2 $kmax 1 $pmax 4 $nmax $runs finger
loop 2 $kmax 1 $pmax 4 $nmax $runs ring
loop 2 $kmax 1 $pmax 4 $nmax $runs xor
