//node 1
package main
import (
	"log"
	"fmt"
	"time"
	"os"
	"strings"
	"strconv"
	"net/rpc"
	"./govec"
//	"sync"
)
// args in get(args)
type GetArgs struct {
	Key    string // key to look up
	VStamp []byte // vstamp(nil)
}

// args in put(args)
type PutArgs struct {
	Key    string // key to associate value with
	Val    string // value
	VStamp []byte // vstamp(nil)
}

// args in testset(args)
type TestSetArgs struct {
	Key     string // key to test
	TestVal string // value to test against actual value
	NewVal  string // value to use if testval equals to actual value
	VStamp  []byte // vstamp(nil)
}

// Reply from service for all three API calls above.
type ValReply struct {
	Val    string // value; depends on the call
	VStamp []byte // vstamp(nil)
}

var mypriority, masterpriority, masterpriorityShouldbe int64
var counter int64
var masterCounter int64
var myid string
var logger (*govec.GoLog)
func main(){
	
	if len(os.Args) != 4 {
		fmt.Printf("Syntax: <Executable> <Ip:port> <id> <logfilename>")
		os.Exit(0)
	}
	
	logger = govec.Initialize("Log"+ os.Args[3], os.Args[3])
	client, err := rpc.Dial("tcp", os.Args[1] )
	if err != nil {
		log.Fatal("dailing:",err)
	}
	//var waitgp sync.WaitGroup
	counter = 1
	masterCounter =0
	masterpriorityShouldbe = 0
	myid = os.Args[2]
	//test if there is priority key entry in the table
	var rply_getPriority ValReply
	var arg GetArgs
	arg.Key = "priority"
	arg.VStamp = logger.PrepareSend("get:priority", nil)
	err = client.Call("KeyValService.Get",&arg, &rply_getPriority)
	if err != nil {
		log.Fatal("KeyValService.Get:", err.Error())
	}
	fmt.Println("Received priority: ", rply_getPriority.Val )
	if rply_getPriority.Val == "" {
		//implies that it is the first node, so set up the table

		// (1) set up priority key
		priority_val := "2"
		var args_setPriority PutArgs
		args_setPriority.Key = "priority"
		args_setPriority.Val = priority_val
		args_setPriority.VStamp = logger.PrepareSend("put:"+ priority_val, nil)
		var reply_setPriority ValReply
		err = client.Call("KeyValService.Put",&args_setPriority, &reply_setPriority)
		if err != nil {
			log.Fatal("KeyValService.Put:", err.Error())
		}
		/*
		if reply_setPriority.Val == "" {
			fmt.Println("Setting priority: success"  )
		}
		*/
		// (2) set up its own entry
		mypriority = 1
		ownEntry_val := strconv.FormatInt(mypriority, 10)+ string(":")+ strconv.FormatInt(counter,10) // add timestamp
		var args2 PutArgs
		args2.Key = os.Args[2]
		args2.Val = ownEntry_val
		args2.VStamp = logger.PrepareSend("put:"+ ownEntry_val, nil)
		var reply_ownEntry ValReply
		err = client.Call("KeyValService.Put",&args2, &reply_ownEntry)
		if err != nil {
			log.Fatal("KeyValService.Put:", err.Error())
		}
		/*
		if reply_ownEntry.Val == "" {
			fmt.Println("Setting own entry: success"  )
		}
		*/

		// (3) set up nodes key
		setNode_val := os.Args[2] + string(":1")
		var args_setNodes PutArgs
		args_setNodes.Key = "nodes"
		args_setNodes.Val = setNode_val
		args_setNodes.VStamp = logger.PrepareSend("put:"+ setNode_val, nil)
		var reply_setNodes ValReply
		err = client.Call("KeyValService.Put",&args_setNodes, &reply_setNodes)
		if err != nil {
			log.Fatal("KeyValService.Put:", err.Error())
		}
		/*
		if reply_setNodes.Val == "" {
			fmt.Println("Setting up value in nodes: success"  )
		}
		*/
		domainMaster(client)

	} else {
		//act as not-first node
		// (1) update the priority key
		mypriority,err = strconv.ParseInt(rply_getPriority.Val,10,32)
		priority_val := strconv.FormatInt(mypriority+1, 10)
		var args_setPriority PutArgs
		args_setPriority.Key = "priority"
		args_setPriority.Val = priority_val
		args_setPriority.VStamp = logger.PrepareSend("put:"+ priority_val, nil)
		var reply_setPriority ValReply
		err = client.Call("KeyValService.Put",&args_setPriority, &reply_setPriority)
		if err != nil {
			log.Fatal("KeyValService.Put:", err.Error())
		}
		/*
		if reply_setPriority.Val == ""{
			fmt.Println("updated priority key: success")
		}
		*/
		regularNode(client)
		for {
			startElection(client)
		}
	}
}

func domainMaster(client *rpc.Client){
	// (4) Accept the role of leader, that is start updating the active key		

	// create a local table for regular node database
	var idtopreviousCounter map[string]int64
	idtopreviousCounter = make(map[string]int64)	

	tickChannel := time.NewTicker(time.Millisecond * 2000).C
	for {
		select {
			case <- tickChannel:
				counter++
				// (4.1) get and set entry in nodes
				var reply_getNodes ValReply
				var arg_getNodes GetArgs
				arg_getNodes.Key = "nodes"
				arg_getNodes.VStamp = logger.PrepareSend("get:nodes", nil)
				err := client.Call("KeyValService.Get",&arg_getNodes, &reply_getNodes)
				if err != nil {
					log.Fatal("KeyValService.Get:", err.Error())
				}
				//fmt.Println("Received nodes: ", reply_getNodes.Val)
				//check if entry is already there in nodes
				//found := false
				var active string
				active = myid + string(":") + strconv.FormatInt(mypriority,10)
				msgparts := strings.Split(reply_getNodes.Val,";;")	
				for i := range msgparts {
					keyVal := strings.Split(msgparts[i],":")	
					// check if the node is still alive or not
					var reply_getNodes ValReply
					var arg_getNodes GetArgs
					arg_getNodes.Key = keyVal[0]
					arg_getNodes.VStamp = logger.PrepareSend("get: "+ keyVal[0], nil)
					err := client.Call("KeyValService.Get",&arg_getNodes, &reply_getNodes)
					if err != nil {
						log.Fatal("KeyValService.Get:", err.Error())
					}
					strcounter := strings.Split(reply_getNodes.Val,":")
					counter,err := strconv.ParseInt(strcounter[1],10,64)
					if err != nil {
						log.Fatal("couldn't parse from string to int:", err.Error())
					}
					if idtopreviousCounter[keyVal[0]] == 0 {
						idtopreviousCounter[keyVal[0]] = counter
					} else {
						if idtopreviousCounter[keyVal[0]] < counter {
							// it alive, append it
							if active != "" {
								active = active + string(";;")
							}
							active = active + keyVal[0] + string(":") + keyVal[1]
							idtopreviousCounter[keyVal[0]] = counter
						} else {
							idtopreviousCounter[keyVal[0]] = 0
						}
					}
					// append alive in active
				}
				fmt.Println("active processes: ", active)	
				if active != "" {
					 //update the active
					setActive_val := active + string(";;;") + myid +string(":") +strconv.FormatInt(mypriority,10) +(":")+strconv.FormatInt(counter,10) //actually it should be active
					var arg_setActive PutArgs
					arg_setActive.Key = "active"
					arg_setActive.Val = setActive_val
					arg_setActive.VStamp = logger.PrepareSend("Put:" + setActive_val ,nil)
					var reply_setActive ValReply
					err = client.Call("KeyValService.Put",&arg_setActive, &reply_setActive)
					if err != nil {
						log.Fatal("KeyValService.Put:", err.Error())
					}
/*					// update nodes
					arg_setActive.Key = "nodes"
					arg_setActive.VStamp = logger.PrepareSend("Put:" + setActive_val ,nil)
					err = client.Call("KeyValService.Put",&arg_setActive, &reply_setActive)
					if err != nil {
						log.Fatal("KeyValService.Put:", err.Error())
					}*/
				}
		}
	}	
}

func regularNode(client * rpc.Client) {
		
	needElection := false
	tickChannel_updatetable := time.NewTicker(1000 * time.Millisecond).C
	tickChannel_pollforActive := time.NewTicker(4000 * time.Millisecond).C
	for {
		select {
			case <- tickChannel_updatetable:
				// (2) set up its own entry
				counter++
				ownEntry_val := strconv.FormatInt(mypriority,10) + string(":") + strconv.FormatInt(counter, 10)  // add timestamp
				var args2 PutArgs
				args2.Key = myid
				args2.Val = ownEntry_val
				args2.VStamp = logger.PrepareSend("put:"+ ownEntry_val, nil)
				var reply_ownEntry ValReply
				err := client.Call("KeyValService.Put",&args2, &reply_ownEntry)
				if err != nil {
					log.Fatal("KeyValService.Put:", err.Error())
				}
				/*
				if reply_ownEntry.Val == ""{
					fmt.Println("set up its own entry: success")
				}
				*/
				// (3) get and set entry in nodes
				var reply_getNodes ValReply
				var arg_getNodes GetArgs
				arg_getNodes.Key = "nodes"
				arg_getNodes.VStamp = logger.PrepareSend("get:nodes", nil)
				err = client.Call("KeyValService.Get",&arg_getNodes, &reply_getNodes)
				if err != nil {
					log.Fatal("KeyValService.Get:", err.Error())
				}
				//fmt.Println("Received nodes: ", reply_getNodes.Val);
				//check if entry is already there in nodes
				var nodes string
				nodes = ""
				found := false
				msgparts := strings.Split(reply_getNodes.Val,";;")	
				for i := range msgparts {
					keyVal := strings.Split(msgparts[i],":")	
					if keyVal[0] == myid  {
						found = true
						val,err := strconv.ParseInt(keyVal[1],10,64)
						if err != nil{
							log.Fatal("couldn't parse key val:", err.Error())
						}
						if val != mypriority {
							if nodes != "" {
								nodes = nodes + string(";;")
							}
							nodes = nodes + keyVal[0] +string(":")+ strconv.FormatInt(mypriority,10)
							
						} else {
							if nodes != "" {
								nodes = nodes + string(";;")
							}
							nodes = nodes + msgparts[i]
						}
					}else {
						if nodes != "" {
							nodes = nodes + string(";;")
						}
						nodes = nodes + msgparts[i]
					}
				}
				var setNodes_val string	
				if found == false {
					// (4) put its node in nodes
					setNodes_val = reply_getNodes.Val + string(";;") + myid + string(":") + strconv.FormatInt(mypriority,10) 
				} else {
					setNodes_val = nodes
				}
				//fmt.Println("setting nodes value to ", setNodes_val)
				var reply_setNodes ValReply
				var arg_setNodes PutArgs
				arg_setNodes.Key = "nodes"
				arg_setNodes.Val = setNodes_val
				arg_setNodes.VStamp = logger.PrepareSend("Put:" + setNodes_val ,nil)
				err = client.Call("KeyValService.Put",&arg_setNodes, &reply_setNodes)
				if err != nil {
					log.Fatal("KeyValService.Put:", err.Error())
				}
				/*
				if reply_setNodes.Val == "" {
					fmt.Println("updated nodes key: success")
				}
				*/
			case <- tickChannel_pollforActive:
				// (4) wait for (t+2) sec and get list of active nodes
				var reply_getActive ValReply
				var arg_getActive GetArgs
				arg_getActive.Key = "active"
				arg_getActive.VStamp = logger.PrepareSend("get:active nodes", nil)
				err := client.Call("KeyValService.Get",&arg_getActive, &reply_getActive)
				if err != nil {
					log.Fatal("KeyValService.Get:", err.Error())
				}
				fmt.Println("active Nodes:", reply_getActive.Val)
				if masterCounter == 0 {
					msgpart := strings.Split(reply_getActive.Val, ";;;")
					masterdetail := strings.Split(msgpart[1], ":")
					masterCounter,err = strconv.ParseInt(masterdetail[2],10,64)
					masterpriority, err = strconv.ParseInt(masterdetail[1],10,64)
					if err != nil {
						log.Fatal("couldnot parse mastercounter:", err.Error())
					}
				}else {
					msgpart := strings.Split(reply_getActive.Val, ";;;")
					masterdetail := strings.Split(msgpart[1], ":")
					masterpriority, err = strconv.ParseInt(masterdetail[1],10,64)
					mCounter,err := strconv.ParseInt(masterdetail[2],10,64)
					if err != nil {
						log.Fatal("couldnot parse mastercounter:", err.Error())
					}
					if mCounter <= masterCounter {
						// need elcetion 
						needElection = true
						fmt.Println("Need Election")
						break
					}
					masterCounter = mCounter
				}

		}
		if needElection == true {
			break
		}
	}
}

func startElection(client *rpc.Client){
	// get list of active processes
	var reply_getActive ValReply
	var arg_getActive GetArgs
	arg_getActive.Key = "active"
	arg_getActive.VStamp = logger.PrepareSend("get:active nodes", nil)
	err := client.Call("KeyValService.Get",&arg_getActive, &reply_getActive)
	if err != nil {
		log.Fatal("KeyValService.Get:", err.Error())
	}
	fmt.Println("active reply:", reply_getActive.Val)

	msgpart := strings.Split(reply_getActive.Val, ";;;")
	var nextleaderid string
	var leaderpriority int64
	if masterpriorityShouldbe == masterpriority || masterpriorityShouldbe == 0  {
		nextleaderid, leaderpriority = probableleader(msgpart[0], masterpriority)
	}else {
		nextleaderid, leaderpriority = probableleader(msgpart[0], masterpriorityShouldbe)
	}
	fmt.Println("nextleaderid: ", nextleaderid, " leaderpriority: ", leaderpriority)
	if nextleaderid == myid{
		//take leader role
		fmt.Println("****Taking leader role****")
		counter = masterCounter
		masterpriority = leaderpriority
		masterpriorityShouldbe = 0
		domainMaster(client)
	}else {
		//continue regular node role
		masterpriorityShouldbe = leaderpriority
		regularNode(client)
	}
}

func probableleader(activeprocesslist string, currentleaderpriority int64) (string, int64){
	//check the id with least priority number
	activeProcesses := strings.Split(activeprocesslist,";;")
	//fmt.Println("Active Process: ",activeProcesses)
	//id of top priority
	var nextleaderpriority int64
	var nextleaderid string
	nextleaderid = ""
	nextleaderpriority = 1000
	for i := range activeProcesses {
		temp := strings.Split(activeProcesses[i], ":")
		id := temp[0]
		priority,err := strconv.ParseInt(temp[1],10,32)
		if err != nil {
			fmt.Println("couldn't parse")
		}
		fmt.Println("id: ", id, " priority: ",priority)
		if priority < nextleaderpriority && priority > currentleaderpriority {
			nextleaderid = id
			nextleaderpriority = priority
		} 
	}
	return nextleaderid, nextleaderpriority
}
