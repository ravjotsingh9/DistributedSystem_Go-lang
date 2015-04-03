// Version 1.0
//
// A simple key-value store that supports three API calls over rpc:
// - get(key)
// - put(key,val)
// - testset(key,testval,newval)
//
// Dependencies:
// - GoVector: https://github.com/arcaneiceman/GoVector

package main

import (
	"./govec"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"time"
	"strings"
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

// Value in the key-val store.
type MapVal struct {
	value  string       // the underlying value representation
	logger *govec.GoLog // GoVector instance for the *key* that this value is mapped to
}

// Map implementing the key-value store.
var kvmap map[string]*MapVal
var port string // port I am running on

// Reserved value in the service that is used to indicate that the key
// is unavailable: used in return values to clients and internally.
const unavail string = "unavailable"

type BackEnd int

// Lookup a key, and if it's used for the first time, then initialize its value.
func lookupKey(key string) *MapVal {
	// lookup key in store
	val := kvmap[key]
	if val == nil {
		// key used for the first time: create and initialize a MapVal instance to associate with a key
		val = &MapVal{
			value:  "",
			logger: govec.Initialize("key-"+key, "key-"+key),
		}
		kvmap[key] = val
		keyCount = keyCount + 1
		c <- keyCount
		fmt.Printf("(in lookup)KeyCount %d\n", keyCount)
	}
	return val
}

// The probability with which a key operation triggers permanent key unavailability.
var failProb float64

// Check whether a key should fail with independent fail probability.
func checkFail(val *MapVal) bool {
	if val.value == unavail {
		return true
	}
	if rand.Float64() < failProb {
		val.value = unavail // permanent unavailability
		return true
	}
	return false
}

// GET
func (kvs *BackEnd) Get(args *GetArgs, reply *ValReply) error {
	val := lookupKey(args.Key)
	val.logger.UnpackReceive("get(k:"+args.Key+")", args.VStamp)

	if checkFail(val) {
		reply.Val = unavail
		reply.VStamp = val.logger.PrepareSend("get-re:"+unavail, nil)
		return nil
	}

	reply.Val = val.value // execute the get
	reply.VStamp = val.logger.PrepareSend("get-re:"+val.value, nil)
	return nil
}

// PUT
func (kvs *BackEnd) Put(args *PutArgs, reply *ValReply) error {
	val := lookupKey(args.Key)
	val.logger.UnpackReceive("put(k:"+args.Key+",v:"+args.Val+")", args.VStamp)

	if checkFail(val) {
		reply.Val = unavail
		reply.VStamp = val.logger.PrepareSend("put-re:"+unavail, nil)
		return nil
	}
	val.value = args.Val // execute the put
	reply.Val = ""
	reply.VStamp = val.logger.PrepareSend("put-re", nil)
	return nil
}

// TESTSET
func (kvs * BackEnd) TestSet(args *TestSetArgs, reply *ValReply) error {
	val := lookupKey(args.Key)
	val.logger.UnpackReceive("testset(k:"+args.Key+",tv:"+args.TestVal+",nv:"+args.NewVal+")", args.VStamp)

	if checkFail(val) {
		reply.Val = unavail
		reply.VStamp = val.logger.PrepareSend("testset-re:"+unavail, nil)
		return nil
	}

	// execute the testset
	if val.value == args.TestVal {
		val.value = args.NewVal
	}

	reply.Val = val.value
	reply.VStamp = val.logger.PrepareSend("testset-re:"+val.value, nil)
	return nil
}

//*************************************
var mypriority, masterpriority, masterpriorityShouldbe int64
var counter int64
var masterCounter int64
var myid string
var logger (*govec.GoLog)
var c = make(chan int64)
var keyCount int64

func nodemain(ip_port string, logfile string){
//	logger = govec.Initialize("Log"+ os.Args[3], os.Args[3])
	logger = govec.Initialize("Log"+ logfile, logfile)
	client, err := rpc.Dial("tcp", ip_port )
	if err != nil {
		log.Fatal("dailing:",err)
	}
	counter = 1
	masterCounter =0
	masterpriorityShouldbe = 0
	//test if there is priority key entry in the table
	var rply_getPriority ValReply
	var arg GetArgs
	arg.Key = "priority"
	arg.VStamp = logger.PrepareSend("get:priority", nil)
	err = client.Call("FrontEndService.Get",&arg, &rply_getPriority)
	if err != nil {
		log.Fatal("FrontEndService.Get:", err.Error())
	}
	fmt.Println("Received priority: ", rply_getPriority.Val )
		//act as not-first node
		// (1) update the priority key
		mypriority,err = strconv.ParseInt(rply_getPriority.Val,10,32)
		priority_val := strconv.FormatInt(mypriority+1, 10)
		var args_setPriority PutArgs
		args_setPriority.Key = "priority"
		args_setPriority.Val = priority_val
		args_setPriority.VStamp = logger.PrepareSend("put:"+ priority_val, nil)
		var reply_setPriority ValReply
		err = client.Call("FrontEndService.Put",&args_setPriority, &reply_setPriority)
		if err != nil {
			log.Fatal("FrontEndService.Put:", err.Error())
		}
		/*
		if reply_setPriority.Val == ""{
			fmt.Println("updated priority key: success")
		}
		*/
		regularNode(client)
}


func regularNode(client * rpc.Client) {
	keyCount =0
	needElection := false
	tickChannel_updatetable := time.NewTicker(250 * time.Millisecond).C
	tickChannel_pollforActive := time.NewTicker(1000 * time.Millisecond).C
	for {
		select {
			case <- c:
					keyCount++
					fmt.Printf("KeyCount %d\n", keyCount)

			case <- tickChannel_updatetable:
				// (2) set up its own entry
				counter++
				ownEntry_val := strconv.FormatInt(mypriority,10) + string(":") + strconv.FormatInt(counter, 10)  // add timestamp
				var args2 PutArgs
				args2.Key = myid
				args2.Val = ownEntry_val
				args2.VStamp = logger.PrepareSend("put:"+ ownEntry_val, nil)
				var reply_ownEntry ValReply
				err := client.Call("FrontEndService.Put",&args2, &reply_ownEntry)
				if err != nil {
					log.Fatal("FrontEndService.Put:", err.Error())
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
				err = client.Call("FrontEndService.Get",&arg_getNodes, &reply_getNodes)
				if err != nil {
					log.Fatal("FrontEndService.Get:", err.Error())
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
							fmt.Printf("->KeyCount %d\n", keyCount)
							nodes = nodes + keyVal[0] +string(":")+ strconv.FormatInt(mypriority,10) + string(":") + strconv.FormatInt(keyCount,10)
							
						} else {
							if nodes != "" {
								nodes = nodes + string(";;")
							}
							//nodes = nodes + msgparts[i]
							nodes = nodes + keyVal[0] +string(":")+ strconv.FormatInt(mypriority,10) + string(":") + strconv.FormatInt(keyCount,10)
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
				fmt.Printf("-->KeyCount %d\n", keyCount)
					setNodes_val = reply_getNodes.Val + string(";;") + myid + string(":") + strconv.FormatInt(mypriority,10) + string(":")+ strconv.FormatInt(keyCount,10)
				} else {
					setNodes_val = nodes
				}
//				fmt.Println("setting nodes value to ", setNodes_val)
				var reply_setNodes ValReply
				var arg_setNodes PutArgs
				arg_setNodes.Key = "nodes"
				arg_setNodes.Val = setNodes_val
				arg_setNodes.VStamp = logger.PrepareSend("Put:" + setNodes_val ,nil)
				err = client.Call("FrontEndService.Put",&arg_setNodes, &reply_setNodes)
				if err != nil {
					log.Fatal("FrontEndService.Put:", err.Error())
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
				err := client.Call("FrontEndService.Get",&arg_getActive, &reply_getActive)
				if err != nil {
					log.Fatal("FrontEndService.Get:", err.Error())
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






// Main server loop.
func main() {
	// parse args
	usage := fmt.Sprintf("Usage: %s <ip:port> <logfileName>\n", os.Args[0])
	if len(os.Args) != 3 {
		fmt.Printf(usage)
		os.Exit(1)
	}

	ip_port := os.Args[1]
	parts:= strings.Split(ip_port,":")
	port = parts[1]
	logfile := os.Args[2]
	//myid = os.Args[2]
	
	/// use port as my id
	myid = port

/*
	arg, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if arg <= 1 {
//		fmt.Printf(usage)
		fmt.Printf("\tPlease select an <id> higher than 1\n")
		os.Exit(1)
	}

	failProb = arg

	// setup randomization
	rand.Seed(time.Now().UnixNano())
*/

	ip_port_frontend := "127.0.0.1:4000"
 	go nodemain(ip_port_frontend  , logfile)

	// setup key-value store and register service
	kvmap = make(map[string]*MapVal)
	kvservice := new(BackEnd)
	rpc.Register(kvservice)
	l, e := net.Listen("tcp", ip_port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	for {
		conn, _ := l.Accept()
		go rpc.ServeConn(conn)
	}
}




