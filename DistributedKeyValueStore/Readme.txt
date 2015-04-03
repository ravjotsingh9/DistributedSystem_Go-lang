================================================
Name:		Ravjot Singh
-----------------------------------------------
Assignment:	Distributed Key-Value Service
================================================

============
Description:	
============
Distributed Key-Value Algorithm Design:
=======================================

There are two type of nodes in a system:
1. FrontEnd: 
   Responsibility of FrontEnd node is to:
      - responding to client queries
      - tracking kv nodes as they fail and join
      - pertitioning the key-value space between the kv nodes.
2. kv-node: 
   Responsibility of kv nodes is to store the key value and respond to get/put/testset.



Basic Idea of Distributed Key-Value:
=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

Keeping in mind the idea of further extension of this assignment, I used code from the previous assignment. Master code is used in the frontEnd node and Regular node code is used in the kv nodes.


WORKING:
A live kv node polls after t sec and updates its entry in the key-value service running in frontEnd node. The key of its entry is its por-number and value is its priority concatenated with a counter.
------------------------------------
| <port> | <priority>:<counterVal> |
------------------------------------

[Note: priority is not much of concern in this assignment, since it always assumes frontend is the master.]


In the same poll kv nodes also update its entry in "nodes" list along with the number of keys it has.
----------------------------------------------------------------------------------------------------------------------
| nodes | <port1>:<priority1>:<numberOfKeys>;;<port3>:<priority6>:<numberOfKeys>;;<port2>:<priority4>:<numberOfKeys> |
----------------------------------------------------------------------------------------------------------------------

FrontEnd node poll after (2 * t) sec, it fetches "nodes" list and check which all kv nodes are alive and adds them to "active" list. FrontEnd stores the previous counter value of each kv node in its local memory. It compares the previous value with the present value and verifies if it has increased. If it is more than the previous val, it delcare the node as alive else removes its entry. 
-----------------------------------------------------------------------------------------------------------------------------------------------
| active | <id1>:<priority1>:<numberOfKeys>;;<id3>:<priority6>:<numberOfKeys>;;<id2>:<priority4>:<numberOfKeys>;;;<id1>:<priority1>:<counter> |
-----------------------------------------------------------------------------------------------------------------------------------------------

Now the important thing here is polling time. For active list, each kv node poll after (2 * (2 * t)) sec ensuring that FrontEnd has atleast polled once before it poll again.


LOAD BALANCING:
For each PUT, the frontEnd node get the list of active kv nodes and read the attribute corresponding to number of keys a kv node already has. After going through all the kv node "numberOfKey" value, it finds out the one with the minimum value and puts the value on that node.
If a kv node fails, the client will attempt to reinsert unavailable keys which will eventually result in distribution of these keys over all alive kv nodes.
If a new kv node joins, frontend will this node untill it reached at comparable "numberOfKey" values. [Drawback is that it put a lot of load on one node.]
Sometime, beacause of delay to update "numberOfKey" value, it happens in bit of unbalanced key distribution over the kv nodes, but it can be easily overcome by shrinking the value of t.

KEY-VALUE TRACK:
FrontEnd node maintains a local keyvalue table which stores all the keys and the ip address of the corresponding nodes where the key is stored. This also put contraint on having duplicate keys on different kv nodes.
If a node becomes unavailable, all keys refering to that node become unavailable.



====================
Running my solution:
====================

Compile my assignment as follows:
	go build frontEnd.go
	go build kvservicemain.go

And then for running:
- To run frontEnd, use below syntax: 
  <Executable> <ip:port> <ip:port> <logfile>

- To run kvnode, use below syntax: 
  <Executable> <ip:port> <logfile>

- To run program, keep govec folder in the same folder 
  where program is running


=============
IDE/Compiler:	
=============
go 1.4

