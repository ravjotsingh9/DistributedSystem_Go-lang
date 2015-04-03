// Version 1.0
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

// Local store to keep track of keys
var localmap map[string]string

// key service towards clients
type KeyValService int

// key service towards internal kv nodes
type FrontEndService string

//node log file
var logfileName string

// Reserved value in the service that is used to indicate that the key
// is unavailable: used in return values to clients and internally.
const unavail string = "unavailable"


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
func (fes * FrontEndService) Get(args *GetArgs, reply *ValReply) error {
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
func (fes * FrontEndService) Put(args *PutArgs, reply *ValReply) error {
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
func (fes *FrontEndService) TestSet(args *TestSetArgs, reply *ValReply) error {
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




// Main kv server loop.Provides service to internal kv nodes
func kvInterface(ip_port string) {

	// setup randomization
	rand.Seed(time.Now().UnixNano())

	kvservice := new(KeyValService)
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

//variable to store ip_port
var ip_p string

// Main kv server loop.Provides service to clients
func frontEndInterface(ip_port string){
	ip_p = ip_port
	// setup key-value store and register service
	kvmap = make(map[string]*MapVal)
	frontservice := new(FrontEndService)
	rpc.Register(frontservice)
	l,e := net.Listen("tcp", ip_port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	for {
		conn, _ := l.Accept()
		go rpc.ServeConn(conn)
	}
}


// Lookup for a kv node available
func lookup() string{
	client, err := rpc.Dial("tcp", ip_p )
	if err != nil {
		log.Fatal("dailing:",err)
	}
	var reply_getActive ValReply
	var arg_getActive GetArgs
	arg_getActive.Key = "active"
	arg_getActive.VStamp = logger.PrepareSend("get:active", nil)
	err = client.Call("FrontEndService.Get",&arg_getActive, &reply_getActive)
	if err != nil {
		log.Fatal("FrontEndService.Get:", err.Error())
	}
	//fmt.Println(">Received nodes: ", reply_getActive.Val)
	msgparts := strings.Split(reply_getActive.Val,";;")	
	var p string
	var c int64
	c = 9999999
	for i := range msgparts {
		keyVal := strings.Split(msgparts[i],":")
		if keyVal[0] == "1"{
			continue
		}
		count,_ := strconv.ParseInt(keyVal[2],10,32)
		if (count < c){
			p = keyVal[0]
			c = count
		}	
	}
	return "127.0.0.1:" + p
}

func localPut(key string, val string) {
	localmap[key] = val
}

// local get
func localGet(key string) string {
	return localmap[key]
}

// GET 
func (kvs * KeyValService) Get(args *GetArgs, reply *ValReply) error {
	ip_port := localGet((*args).Key)
	fmt.Println("Get val for key "+ (*args).Key + " from " + ip_port)
	if ip_port == ""{
		reply.Val = ""
		reply.VStamp = logger.PrepareSend("get-re: ", nil)
		return nil
	} else{
		client, err := rpc.Dial("tcp", ip_port )
		if err != nil {
			localPut((*args).Key,"")
			reply.Val = unavail
			reply.VStamp = logger.PrepareSend("get-re: unavail", nil)
			return nil
			//log.Fatal("dailing:",err)
		}
		var bargs GetArgs
		bargs.Key = (*args).Key
		bargs.VStamp = (*args).VStamp
		var breply ValReply
		err = client.Call("BackEnd.Get",&bargs, &breply)
		if err != nil {
			localPut((*args).Key,"")
			reply.Val = unavail
			reply.VStamp = logger.PrepareSend("get-re: unavail", nil)
			return nil
		}
		(*reply).Val = breply.Val
		(*reply).VStamp = breply.VStamp
	}
		return nil
}

// PUT
func (kvs *KeyValService) Put(args *PutArgs, reply *ValReply) error {
	ip_port := localGet((*args).Key)
	if ip_port == ""{
		ip_port = lookup()
	}
	fmt.Println("Put val for key " + (*args).Key  + " in " + ip_port)
	client, err := rpc.Dial("tcp", ip_port )
	if err != nil {
			reply.Val = unavail
			reply.VStamp = logger.PrepareSend("get-re: unavail", nil)
			return nil
		//log.Fatal("dailing:",err)
	}

	var breply ValReply
	err = client.Call("BackEnd.Put",args, &breply)
	if err != nil {
		localPut((*args).Key,"")
		reply.Val = unavail
		reply.VStamp = logger.PrepareSend("get-re: unavail", nil)
		return nil
	}
	(*reply).Val = breply.Val
	(*reply).VStamp = breply.VStamp
	localPut((*args).Key, ip_port)
	return nil
}

// TESTSET
func (kvs *KeyValService) TestSet(args *TestSetArgs, reply *ValReply) error {
	ip_port := localGet((*args).Key)
	if ip_port == ""{
		ip_port = lookup()
	}
	fmt.Println("TestSet val for key " + (*args).Key  + " in " + ip_port)
	client, err := rpc.Dial("tcp", ip_port )
	if err != nil {
			reply.Val = unavail
			reply.VStamp = logger.PrepareSend("get-re: unavail", nil)
			return nil
		//log.Fatal("dailing:",err)
	}
	var breply ValReply
	err = client.Call("BackEnd.TestSet",args, &breply)
	if err != nil {
		localPut((*args).Key,"")
		reply.Val = unavail
		reply.VStamp = logger.PrepareSend("get-re: unavail", nil)
		return nil
	}
	(*reply).Val = breply.Val
	(*reply).VStamp = breply.VStamp
	return nil
}

//************************************ 
var mypriority, masterpriority, masterpriorityShouldbe int64
var counter int64
var masterCounter int64
var myid string
var logger (*govec.GoLog)
func nodemain(ip_port string, logfile string){
	logger = govec.Initialize("Log"+ logfile, logfile)
	client, err := rpc.Dial("tcp", ip_port )
	if err != nil {
		log.Fatal("dailing:",err)
	}
	counter = 1
	masterCounter =0
	masterpriorityShouldbe = 0
	//test if there is priority key entry in the table
		//implies that it is the first node, so set up the table

		// (1) set up priority key
		priority_val := "2"
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
		if reply_setPriority.Val == "" {
			fmt.Println("Setting priority: success"  )
		}
		*/
		// (2) set up its own entry
		mypriority = 1
		ownEntry_val := strconv.FormatInt(mypriority, 10)+ string(":")+ strconv.FormatInt(counter,10) // add timestamp
		var args2 PutArgs
		args2.Key = myid
		args2.Val = ownEntry_val
		args2.VStamp = logger.PrepareSend("put:"+ ownEntry_val, nil)
		var reply_ownEntry ValReply
		err = client.Call("FrontEndService.Put",&args2, &reply_ownEntry)
		if err != nil {
			log.Fatal("FrontEndService.Put:", err.Error())
		}
		/*
		if reply_ownEntry.Val == "" {
			fmt.Println("Setting own entry: success"  )
		}
		*/

		// (3) set up nodes key
		setNode_val := myid + string(":1")
		var args_setNodes PutArgs
		args_setNodes.Key = "nodes"
		args_setNodes.Val = setNode_val
		args_setNodes.VStamp = logger.PrepareSend("put:"+ setNode_val, nil)
		var reply_setNodes ValReply
		err = client.Call("FrontEndService.Put",&args_setNodes, &reply_setNodes)
		if err != nil {
			log.Fatal("FrontEndService.Put:", err.Error())
		}
		/*
		if reply_setNodes.Val == "" {
			fmt.Println("Setting up value in nodes: success"  )
		}
		*/
		domainMaster(client)
}

func domainMaster(client *rpc.Client){
	// (4) Accept the role of leader, that is start updating the active key		

	// create a local table for regular node database
	var idtopreviousCounter map[string]int64
	idtopreviousCounter = make(map[string]int64)	

	tickChannel := time.NewTicker(time.Millisecond * 500).C
	for {
		select {
			case <- tickChannel:
				counter++
				// (4.1) get and set entry in nodes
				var reply_getNodes ValReply
				var arg_getNodes GetArgs
				arg_getNodes.Key = "nodes"
				arg_getNodes.VStamp = logger.PrepareSend("get:nodes", nil)
				err := client.Call("FrontEndService.Get",&arg_getNodes, &reply_getNodes)
				if err != nil {
					log.Fatal("FrontEndService.Get:", err.Error())
				}
				fmt.Println("Received nodes: ", reply_getNodes.Val)
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
					err := client.Call("FrontEndService.Get",&arg_getNodes, &reply_getNodes)
					if err != nil {
						log.Fatal("FrontEndService.Get:", err.Error())
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
							active = active + keyVal[0] + string(":") + keyVal[1] + string(":") + keyVal[2]
							idtopreviousCounter[keyVal[0]] = counter
						} else {
							idtopreviousCounter[keyVal[0]] = 0
						}
					}
					// append alive in active
				}
				fmt.Println("active processes: ", active)
				// put active key in local map
				localmap["active"] = active	

				if active != "" {
					 //update the active
					setActive_val := active + string(";;;") + myid +string(":") +strconv.FormatInt(mypriority,10) +(":")+strconv.FormatInt(counter,10) //actually it should be active
					var arg_setActive PutArgs
					arg_setActive.Key = "active"
					arg_setActive.Val = setActive_val
					arg_setActive.VStamp = logger.PrepareSend("Put:" + setActive_val ,nil)
					var reply_setActive ValReply
					err = client.Call("FrontEndService.Put",&arg_setActive, &reply_setActive)
					if err != nil {
						log.Fatal("FrontEndService.Put:", err.Error())
					}
/*					// update nodes
					arg_setActive.Key = "nodes"
					arg_setActive.VStamp = logger.PrepareSend("Put:" + setActive_val ,nil)
					err = client.Call("FrontEndService.Put",&arg_setActive, &reply_setActive)
					if err != nil {
						log.Fatal("FrontEndService.Put:", err.Error())
					}*/
				}
		}
	}	
}


//************************************

/**
*	Function Name: main
*	Desc : start function
*/

func main() {
	// parse args
	usage := fmt.Sprintf("Usage: %s <client-interfacing ip:port> <kv-interfacing ip:port> <logfile> \n", os.Args[0])
	if len(os.Args) != 4 {
		fmt.Printf(usage)
		os.Exit(1)
	}

	ip_port_kvInterface := os.Args[2]

	ip_port_clientInterface := os.Args[1]

	logfileName = os.Args[3]
	myid = "1"

	localmap = make(map[string]string)
	go frontEndInterface(ip_port_kvInterface)

	go kvInterface(ip_port_clientInterface)
	time.Sleep(1 * time.Second)	
	nodemain(ip_port_kvInterface, logfileName)

}
