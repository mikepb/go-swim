# go-swim

The `go-swim` package implements a the [SWIM failure detector][swim] with a focus on rapid protocol prototyping. Specifically, the basic SWIM failure detection algorithm, described in a later section, is decomposed into the node selector component and the ping mechanism to allow us to replace the former with topologically-defined variants. The design of `go-swim` is inspired by and borrows heavily from HashiCorp's [memberlist][] implementation of SWIM.


## Special thanks to our sponsors

![HashiCorp](docs/hashicorp_logo.png)

This project is generously sponsored by [HashiCorp][hashicorp]. We especially thank Mitchell Hashimoto and Armon Dadgar for their valuable feedback on developing the research goals for `go-swim`.

## Usage

TBD


## Configuration parameters

`T` is the protocol period

`S` is the suspicion period

`p` is the number of local peers that a processes uses for direct pings

`k` is the number of regional peers that a process uses for batched indirect pings

`d(x)` is the distance metric

`r` is the radius, the number of peers a process considers to be inside its neighborhood

`s` is the number of neighboring peers that a process pings each protocol period


## The SWIM failure detection algorithm

SWIM solves the problem of providing each member process in a group a list of all non-faulty member processes, updating the list as processes join, leave, and fail. The algorithm consists of two components: the *failure detector* and the *disseminator*. The failure detector is responsible for detecting when the membership status of a process changes. The disseminator is responsible for updating all member processes of changes in the member list.

The failure detection consists of two sub-protocols: basic SWIM and the _suspicion_ sub-protocol. Basic SWIM detects possible process failures, labeling these processes as _suspect_. When a process is marked as suspect, the _suspicion_ sub-protocol gives the suspect process time to refute its failure, reducing the probability of false positives. The combined failure detection protocol is run every _protocol period_ of _T_ time units, where _T_ is a configurable parameter that sets the interval between iterations of the failure detection algorithm.

At the start of every protocol period, processes run the basic SWIM sub-protocol, in which a process `m_i` pings `p` target processes. In the original SWIM paper, `p=1`. If a target process `m_j` receives a ping, but fails to respond in a timely fashion, `m_i` asks `k` unrelated processes to ping `m_j`. If `m_j` does not respond to either pings before the start of the next ping period, the suspicion sub-protocol is initiated for `m_j`: the process `m_i` marks `m_j` as suspicious and disseminates the update to the group. If no process disputes the status of `m_j` before the end of the suspicion period, through disseminating the appropriate status update, the group marks it as failed. The suspicion period is calculated from the protocol period, the number of processes in the group, and a small multiplier.

To send all group members process status updates, SWIM implements piggybacked gossip dissemination, in which each process sends the updates to processes as extra data attached to the monitoring pings. As such, the method used to select which process to ping determines the time to disseminate updates to all group members.

The original SWIM failure detector used the round-robin selection method, shuffling the list of processes after each round. This randomization method allows SWIM to detect suspected failures in amortized ~1.58 and worst-case `(2n - 1)/k` protocol periods, where `n` is the number of processes in the group and `k` the number of processes to ping each protocol period. Likewise, the method achieves amortized _O(log n)_ and worst-case _O(n)_ growth in the time for all processes to learn of a change in membership, with respect to the number of processes.

For a more detailed description of the algorithm, please see the original [SWIM][swim] paper.


### Changes from memberlist and SWIM

Memberlist improves on SWIM by introducing join and leave intents, allowing for non-piggybacked gossip, and implementing periodic full state synchronization. `go-swim` similarly uses join and leave intents, respectfully, to totally order a node's membership events from the time it joins and to sidestep the suspicion mechanism when a node gracefully leaves. However, `go-swim` does not implement non-piggybacked gossip nor full state synchronization.

Instead of implementing non-piggybacked gossip outside of the SWIM messages, `go-swim` exposes the `p` configuration parameter to allow nodes to ping `p` other nodes instead of just one. This has the effect of improving both the dissemination and failure detection times at the cost of sending more messages. A future update to `go-swim` may implement allow for varying `p` with the number of pending unsent gossip messages.


## Design documents

- [Architecture][architecture] describes how the data structures in `go-swim` interact to implement the SWIM failure detector.
- [Simulator][simulator] describes the in-process network simulator and how to run it.


## License

`go-swim` is dual-licensed under the Apache 2.0 and Mozilla Public License 2.0 licenses. You use the software under either license, but must comply with the terms of the license you choose. You should have received the text of the licenses as part of the source distribution of the software.


### Mozilla Public License 2.0

> This Source Code Form is subject to the terms of the Mozilla Public
> License, v. 2.0. If a copy of the MPL was not distributed with this
> file, You can obtain one at http://mozilla.org/MPL/2.0/.


### Apache 2.0

> Copyright 2015 Michael Phan-Ba &lt;michael@mikepb.com&gt;
>
> Licensed under the Apache License, Version 2.0 (the "License");
> you may not use this file except in compliance with the License.
> You may obtain a copy of the License at
>
> http://www.apache.org/licenses/LICENSE-2.0
>
> Unless required by applicable law or agreed to in writing, software
> distributed under the License is distributed on an "AS IS" BASIS,
> WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
> See the License for the specific language governing permissions and
> limitations under the License.


[architecture]: docs/architecture.md
[hashicorp]: https://www.hashicorp.com
[hashios]: https://github.com/hashicorp
[memberlist]: https://github.com/hashicorp/memberlist
[simulator]: docs/simulator.md
[swim]: http://www.cs.cornell.edu/~asdas/research/dsn02-SWIM.pdf
