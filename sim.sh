#!/usr/bin/env bash

# GOMAXPROCS=8 ./sim.sh

go build -o simulate sim/main.go

echo "n	detection delay	broadcast delay	# buckets	# direct pings	metric"

for ((p=1;p<=2;p=p+p)); do
	for ((n=4;n<=64;n=n+n)); do
		for ((i=1;i<=16;i++)); do
			./simulate -n $n -k 1 -p $p || exit
		done
	done
done

for ((k=2;k<=8;k=k+k)); do
	for ((p=1;p<=2;p=p+p)); do
		for ((n=4;n<=64;n=n+n)); do
			for ((i=1;i<=16;i++)); do
				./simulate -n $n -k $k -p $p -d ring || exit
			done
		done
	done
done

for ((k=2;k<=8;k=k+k)); do
	for ((p=1;p<=2;p=p+p)); do
		for ((n=4;n<=64;n=n+n)); do
			for ((i=1;i<=16;i++)); do
				./simulate -n $n -k $k -p $p -d xor || exit
			done
		done
	done
done
