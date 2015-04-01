#!/usr/bin/env bash

# GOMAXPROCS=8 ./sim.sh

kmax=8
pmax=2
nmax=128
imax=32

go build -o simulate sim/main.go

trap 'exit' SIGHUP SIGINT SIGTERM

function simulate {
	isdone=false
	while ! $isdone; do
		./simulate -r 1 -n $1 -k $2 -p $3 -d $4
		isdone=true
	done
}

echo "n	detection delay	broadcast delay	# buckets	# direct pings	metric"

for ((p=1;p<=$pmax;p=p+p)); do
	for ((n=4;n<=$nmax;n=n+n)); do
		for ((i=1;i<=$imax;i++)); do
			simulate $n 1 $p none
		done
	done
done

for ((k=2;k<=8;k=k+k)); do
	for ((p=1;p<=$pmax;p=p+p)); do
		for ((n=4;n<=$nmax;n=n+n)); do
			for ((i=1;i<=$imax;i++)); do
				simulate $n $k $p finger
			done
		done
	done
done

for ((k=2;k<=8;k=k+k)); do
	for ((p=1;p<=$pmax;p=p+p)); do
		for ((n=4;n<=$nmax;n=n+n)); do
			for ((i=1;i<=$imax;i++)); do
				simulate $n $k $p ring
			done
		done
	done
done

for ((k=2;k<=8;k=k+k)); do
	for ((p=1;p<=$pmax;p=p+p)); do
		for ((n=4;n<=$nmax;n=n+n)); do
			for ((i=1;i<=$imax;i++)); do
				simulate $n $k $p xor
			done
		done
	done
done
