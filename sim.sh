#!/usr/bin/env bash

go build -o simulate sim/main.go

echo "n	detection delay	broadcast delay	# buckets	# direct pings	metric"

for ((p=1;p<=2;p++)); do
	for ((n=3;n<=32;n++)); do
		for ((i=1;i<=32;i++)); do
			GOTRACEBACK=2 GOMAXPROCS=8 ./simulate -n $n -k 1 -p $p || exit
		done
	done
done

for ((k=2;k<=2;k++)); do
	for ((p=1;p<=2;p++)); do
		for ((n=3;n<=32;n++)); do
			for ((i=1;i<=32;i++)); do
				GOTRACEBACK=2 GOMAXPROCS=8 ./simulate -n $n -k $k -p $p -d ring || exit
			done
		done
	done
done

for ((k=2;k<=2;k++)); do
	for ((p=1;p<=2;p++)); do
		for ((n=3;n<=32;n++)); do
			for ((i=1;i<=32;i++)); do
				GOTRACEBACK=2 GOMAXPROCS=8 ./simulate -n $n -k $k -p $p -d ring || exit
			done
		done
	done
done
