//Master
//./masterSlave_TimeSync -m 300 500 slaves.txt  LogMaster

//Slave
//./masterSlave_TimeSync -s 127.0.0.1:5000 500 LogSlave

package main

import (
	"./govec"
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
)

type timer struct{
	mutex sync.Mutex
	timenow int64
}

var  gamma int64
var timeNow timer
var Logger (*govec.GoLog)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Syntax to run as master: <Executable> -m <time offset> <d> <slavesfile> <logfile>")
		fmt.Println("Syntax to run as slave: <Executable> -s <ip:port> <time offset> <logfile>")
	} else {
		if os.Args[1] == "-m" {
			if len(os.Args) == 6 {
				mainMaster()
			} else {
				fmt.Println("Syntax: <Executable> -m <time offset> <d> <slavesfile> <logfile>")
			}
		} else if os.Args[1] == "-s" {
			if len(os.Args) == 5 {
				mainSlave()
			} else {
				fmt.Println("Syntax: <Executable> -s <ip:port> <time offset> <logfile>")
			}
		} else {
			fmt.Println("Syntax to run as master: <Executable> -m <time offset> <d> <slavesfile> <logfile>")
			fmt.Println("Syntax to run as slave: <Executable> -s <ip:port> <time offset> <logfile>")
		}
	}
}


/*
 * Main master function
 */


func mainMaster() {
	
	/* validate time offset and save it in a variable */
	timeNw, err := strconv.ParseInt(os.Args[2], 10, 64)
	checkError(err)
	if timeNw >= 0 {
		//timeNow = timeNw
		updateClock(timeNw)
	} else {
		fmt.Println("<time offset> can not be negative")
		return
	}
	
	/* validate gamma(threshold) and save it in a variable */
	gm, err := strconv.ParseInt(os.Args[3], 10, 64)
	checkError(err)
	if gm >= 0 {
		gamma = gm
	} else {
		fmt.Println("<d> can not be negative")
		return
	}
	
	/* Initialize logger */
	Logger = govec.Initialize("Master", os.Args[5])

	/* save slave file name*/
	slavesfile := os.Args[4]
	
	/* Read file */
	addresses := readFile2(slavesfile)
	
	
	//tickChan := time.NewTicker(time.Millisecond * 1000).C
	
	/* independent clock */
	go startClock(1000)
	
	/* independent perodic correction trigger */
	tickChan := time.NewTicker(time.Millisecond * 10000).C
	doneChan := make(chan bool)
	for {
		select {
		/*
		case <-tickChan:
			// increment the time
			timeNow++
			fmt.Println("Ticker ticked ", timeNow)
			*/
		case <-tickChan:
			/* declare a dictionary database */
			var deltaDatabase map[string]int64
			deltaDatabase = make(map[string]int64)
			var correct, sum, count int64
			
			/* put address:delta in dictionary */
			for _, addr := range addresses {
				//fmt.Println(addr)
				var delta int64
				delta = commToSlave(addr)
				if delta != 0 {
					deltaDatabase[addr] = delta
					//fmt.Println("Added:", addr, ":", delta)
				}
			}
			
			/* Correction AVerage calculation */
			count = 1
			sum = 0
			for k := range deltaDatabase {
				val := deltaDatabase[k]
				if val < 0 {
					val = -val
				}
				if val < gamma && val > 0 {
					count++
					sum = sum + deltaDatabase[k]
					//fmt.Println(sum, ":", count)
				}
			}
			if count > 0 && sum !=0 {
				correct = sum / count
				/* do correction in master */
				updateClock( timeNow.timenow - correct)
				fmt.Println("Corrected myself by ", -correct + 1)
			} else if count == 0 {
				correct = 0
			}
			
			/* calculate difference and send correction to slaves */
			for key := range deltaDatabase {
				sendCorr((deltaDatabase[key] - correct), key)
			}
		/* trying to figure out the termination of process */
		case <-doneChan:
			return
		}
	}

}

/*
 * Function to read file
 */

//Obsolete: since it is system dependent
func readFile(file string) (content []string) {
	contents, _ := ioutil.ReadFile(file)
	con := string(contents)
	contentArr := strings.Split(con, "\r\n")
	return contentArr
}

/*
 * Function to read file
 */

func readFile2(file string) (contnts []string) {
	var contents []string
	
	/* open file */
	f, err := os.Open(file)
	if err != nil {
		fmt.Println("error opening file ", err)
		os.Exit(1)
	}
	
	defer f.Close()
	
	/* read file into contents */
	r := bufio.NewReader(f)
	for {
		content, _, err := r.ReadLine()
		//fmt.Println(string(content[0:]))
		if err == io.EOF {
			break
		} else if err != nil {
			return
		}
		contents = append(contents, string(content))
	}
	return contents
}

/*
 * Function for master to communicate slaves
 */


func commToSlave(service string) int64 {
	
	/* Resolve address */
	udpAddr, err := net.ResolveUDPAddr("udp", service)
	if err != nil  {
		//fmt.Println("Error: while resolving address")
		return 0
	}
	
	/* Dial connection to master */
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		//fmt.Println("Error: encountered while dialing connection")
		return 0
	}
	
	/* prepare request */
	//daytime := timeNow
	daytime := timeNow.timenow
	req := string(service) + string(";;") + strconv.FormatInt(daytime, 10)
	
	/* write req to udp connection */
	_, err = conn.Write(Logger.PrepareSend("Req TimeStamp", []byte(req)))
	if err != nil {
		//fmt.Println("Error:encountered while writing to connection ")
		return 0
	}
	
	/* set time out deadline */
	err = conn.SetDeadline(time.Now().Add(100 * time.Millisecond))
	checkError(err)
	
	/* read response from connection */
	var buffer [1024]byte
	n,_, err := conn.ReadFromUDP(buffer[0:])
	if err != nil  {
		//fmt.Println("Error: reading from connection")
		return 0
	}
	
	/* unpack the received response */
	buf := Logger.UnpackReceive("Resp Received", buffer[0:n])
	
	/* close connection */
	conn.Close()

	/* note down the time of reception of response*/
	receivetime := timeNow.timenow
	
	/* calculate delta for this slave and return */
	val := calculateDelta(string(buf), receivetime)
	return val
}

/*
 * Function to calculate delta
 */


func calculateDelta(msg string, receivetime int64) int64 {
	msgparts := strings.Split(msg, ";;")
	
	/* C(t1) */
	t1, err := strconv.ParseInt(msgparts[1], 10, 64)
	checkError(err)
	//fmt.Println("t1: ", t1)
	
	/* C(t2) */
	t2, err := strconv.ParseInt(msgparts[2], 10, 64)
	checkError(err)
	//fmt.Println("t2: ", t2)
	
	/* C(t3) */
	t3 := receivetime
	//fmt.Println("t3: ", t3)
	
	/* if replied on previous round of req, ignore it*/
	if (t3 - t1) > 10 {
		return 0
	}
	
	/* calculate delta */
	delta := ((t1 + t3) / 2) - t2

	//fmt.Println("delta:", strconv.FormatInt(delta, 10))
	return delta
}

/*
 * Function to send correction to the slaves
 */

func sendCorr(val int64, service string) {
	
	/* resolev address */
	udpAddr, err := net.ResolveUDPAddr("udp", service)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	
	/* dial connection to slave */
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	
	/* prepare message */
	corr := "Corr" + string(";;") + string(service) + string(";;") + strconv.FormatInt(val, 10)
	
	/* write message on connection */
	_, err = conn.Write(Logger.PrepareSend("Send Correction", []byte(corr)))
	checkError(err)
	fmt.Println("Correction Sent to ",service, ":", val)
	
	/* close connection */
	conn.Close()
}


/*
 * Main slave function
 */


func mainSlave() {

	/* validate time offset and save it in a variable */
	timeNw, err := strconv.ParseInt(os.Args[3], 10, 64)
	checkError(err)
	if timeNw >= 0 {
		updateClock(timeNw)
	} else {
		fmt.Println("<time offset> can not be negative")
		return
	}
	
	/* Initialize logger */
	Logger = govec.Initialize("Slave", os.Args[4])
	
	/* start clock */
	go startClock(500)
	 /* wait for master */
	waitForMaster(os.Args[2])
}


/*
 * function to start clock for slave
 */
 
func startClock(interval time.Duration) {
	tickChan := time.NewTicker(time.Millisecond * interval).C
	for {
		<-tickChan
		updateClock(timeNow.timenow + 1)
		fmt.Println("Ticker ticked ", timeNow.timenow)
	}
}

/*
 * Function to connect to master
 */

func waitForMaster(service string) {
	udpAddr, err := net.ResolveUDPAddr("udp", service)
	checkError(err)

	conn, err := net.ListenUDP("udp", udpAddr)
	checkError(err)
	for {
		respondMaster(conn)
	}
}

/*
 * Function to handle communication from master
 */


func respondMaster(conn *net.UDPConn) {
	var buffer [1024]byte
	n, addr, err := conn.ReadFromUDP(buffer[0:])
	if err != nil {
		return
	}
	buf := Logger.UnpackReceive("Req Received", buffer[0:n])
	msg := string(buf[0:])
	msgparts := strings.Split(msg, ";;")
	if msgparts[0] == "Corr" {
		val, err := strconv.ParseInt(msgparts[2], 10, 64)
		checkError(err)
		fmt.Println("Corrected by ", val)
		updateClock(timeNow.timenow + val)
	} else {
		daytime := timeNow.timenow
		resp := string(buf[0:]) + string(";;") + strconv.FormatInt(daytime, 10)
		conn.WriteToUDP(Logger.PrepareSend("Resp Timestamp", []byte(resp)), addr)
	}
}


func updateClock(val int64){
	timeNow.mutex.Lock()
	defer timeNow.mutex.Unlock()
	timeNow.timenow = val 
}


/*
 * Function to check error
 */

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}
