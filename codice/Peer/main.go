/* ************************* *
 * Peer for mutual exclusion *
 * ************************* */

package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

type Peer_Api int

func main() {

	fmt.Println("Peer client is up")

	//generate random port for the process
	rand.Seed(987654321 * time.Now().UnixNano())
	n := rand.Intn(9999)

	//init again random generator for delays
	rand.Seed(time.Now().UnixNano())

	//init configuration variables
	var peer_debug bool
	var peer_logger *log.Logger
	var c Conf
	peer_debug = false
	peer_logger = log.Default()

	//check if verbose mode is enabled and init log file if necessary
	if os.Getenv("VERBOSE") == "1" {
		peer_debug = true
		fmt.Println("Debug mode is enabled, program log can be found in /log/Peer" + strconv.Itoa(n) + ".log")
		var err error
		peer_logger, err = InitLogger("Peer" + strconv.Itoa(n))
		if err != nil {
			log.Fatalln("Logging file cannot be created: ", err)
		}
		peer_logger.Println("Peer client starting in debug mode")
	}

	//read conf file and init configuration variables
	c.readConf(peer_logger, peer_debug)
	port := c.PeerPort + n
	c.PeerPort = port
	c.PeerIP = GetOutboundIP()

	//try connecting with registration service
	if peer_debug {
		peer_logger.Println("I'll trying to access shared resources")
	}
	client, err := rpc.DialHTTP("tcp", c.RegIP+":"+strconv.Itoa(c.RegPort))
	try := 0
	for err != nil && strings.Contains(err.Error(), "connection refused") && try < 10 {
		//if the port is closed on first try, try again. Ten tries are allowed
		client, err = rpc.DialHTTP("tcp", c.RegIP+":"+strconv.Itoa(c.RegPort))
	}
	if err != nil && !strings.Contains(err.Error(), "connection refused") {
		if peer_debug {
			peer_logger.Println("Registration service cannot be reached with error: ", err)
		}
		log.Fatalln("Registration service cannot be reached with error: ", err)
	}

	//send request to join mutual exclusion group
	var reply Registration_reply
	myAddress := &Peer{c.PeerIP, c.PeerPort}
	msg_delay()
	err = client.Call("RegistrationApi.CanIJoin", &myAddress, &reply)
	if err != nil {
		if peer_debug {
			peer_logger.Println("Request to join cannot be send: ", err)
		}
		log.Fatalln("Request to join cannot be send: ", err)
	}
	if reply.Alg == NULL {
		//if there are to many peers, exit
		if peer_debug {
			peer_logger.Println("Cannot join mutual exclusion group: too many peers")
		}
		log.Fatalln("To many peers, try again later")
	} else {
		//if registration have been succesful
		peer_logger.Println("Registered!")
		peer := make([]Peer, len(reply.Peer))
		for i, element := range reply.Peer {
			peer[i] = element
		}
		//start mutual exclusion with the correct algorithm
		if reply.Alg == AUTH {
			//authentication with ricart agrawala algorithm
			RicartAgrawala(reply.Index, c, peer, peer_logger, peer_debug)
		} else if reply.Alg == TOKEN {
			//client for centralized token algorithm
			Slave(reply.Index, c, peer, peer_logger, peer_debug)
		} else if reply.Alg == QUORUM {
			//quorum with maekawa algorithm
			mask := make([]int, len(reply.Mask))
			for i, element := range reply.Mask {
				mask[i] = element
			}
			Maekawa(reply.Index, c, peer, mask, peer_logger, peer_debug)
		}
	}
}
