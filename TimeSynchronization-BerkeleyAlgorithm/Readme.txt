=========================================================
Name:		Ravjot Singh
---------------------------------------------------------
Assignment:	Time Synchronization (Berkeley algorithm)
=========================================================

============
Description:	
============
The assignment demonstrate the use of Berkeley algorithm for synchronizing time 
among multiple processes running with different tick.0
My assignment solution consists of the following files:
- masterSlave_TimeSync.go
- slaves.txt (sample input file for master)
- Readme.txt

=============
Prerequisite
=============
The program uses open source GoVector library to track the partial ordering
between important event in the system.
Download it from here : https://github.com/arcaneiceman/GoVector/

====================
Running my solution:
====================

Compile my assignment as follows:
	go build masterSlave_TimeSync.go

And then for running:
- As Master, use below syntax: 
  <Executable> -m <time offset> <d> <slavesfile> <logfile>

- As Slave, use below syntax: 
  <Executable> -s <ip:port> <time offset> <logfile>

- To run program, keep govec folder in the same folder 
  where program is running

Note: Run 1 master and as many slaves as you want.

=============
IDE/Compiler:	
=============
go 1.4


====================
Conclusions/Remarks:
====================
- The tick time is assumed to be 1000ms for master and 500ms for slave.