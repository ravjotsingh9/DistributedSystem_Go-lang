=======================================
Name:		Ravjot Singh
---------------------------------------
Assignment:	Leader Election
=======================================

============
Description:	
============

Basic Idea of Leader Selection:
=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
There are two types of nodes in a system:
1. Master: 
   Responsibility of Master node is to update the list of Active Reguler nodes.
2. Regular: 
   Role of Regular node is to update its entry in the kv-Service as well as fetch the recent list of Active
   Regular nodes advertised by Master.
   The other responsibilty of regular node is that to find out when Master died and elect a leader.


While joining the domain, all nodes are provided with a priority number. At any time, the node with lowest priority number must take on as leader/master.
A live node polls after t sec and updates its entry. The key of its entry is its id and value is its priority concatenated with a counter.
------------------------------------
| <id> | <priority>:<counterVal> |
------------------------------------

In the same poll node also update its entry in "nodes" list.
-------------------------------------------------------------------
| nodes | <id1>:<priority1>;;<id3>:<priority6>;;<id2>:<priority4> |
-------------------------------------------------------------------

Master poll after (2 * t) sec, it fetches "nodes" list and check which all nodes are alive and adds them to "active" list. Master stores the previous counter value of each node in its local memory. It compares the previous value with the present value and verifies if it has increased. If it is more than the previous val, it delcare the node as alive else removes its entry. 
--------------------------------------------------------------------------------------------------
| active | <id1>:<priority1>;;<id3>:<priority6>;;<id2>:<priority4>;;;<id1>:<priority1>:<counter> |
--------------------------------------------------------------------------------------------------
Apart from this, master concatenates its id, priority and counter value at the end of "active" list.

To track if master is alive or not, regular nodes keep track of counter of master and each time they fetach active list, they get to know if master is alive or not. 
Now the important thing here is polling time. For active list, each node poll after (2 * (2 * t)) sec ensuring that master has atleast polled once before it poll again.



=========
CheckList
=========
=> All nodes run the same code and communicate only indirectly, through the key-value service

 - All nodes are running the same code and are not directly communicating at all.

   
=> Given a sufficiently long time during which failures do not occur, an active node is eventually elected as
    a leader.

 - Leader election is properly working.


=> Given a sufficiently long time during which failures do not occur, the elected leader will eventually         
   advertise an accurate list of all active nodes in the system. And, each active node will retrieve the
   latest version of this list from the key-value service.

 - Leader successfully advertise the list of all active nodes in the system identified by the unique id.
 - Each regular retrives the published list periodically.


=> Your implementation must be robust to node halting failures, including leader halting failures.

 - Program is robust to halting failure, including leader failure.


=> Your implementation must be robust to nodes that restart (i.e., halt and later re-join the system with the
   same identity).

 - Program is robust to nodes restart.


=> Your implementation must be robust to varying RPC times.

 - Program checks if RPC call fails.


=> You cannot change the implementation of the key-value service.

 - No changes are made in Key-value service

=> You must use the GoVector library to track the partial ordering between important events (e.g., message
   sends/receives and synchronization events) in your distributed system and to log these events to a log
   file.

 - Used GoVector library for messages.

 
====================
Running my solution:
====================

Compile my assignment as follows:
	go build node.go

And then for running:
- To run, use below syntax: 
  <Executable> <ip:port> <unique Id> <logfile>



=============
IDE/Compiler:	
=============
go 1.4

=======
Remarks
=======
- To run program, keep govec folder in the same folder where program is running