package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

func main() {
	var peer_debug bool
	var peer_logger *log.Logger
	var c Conf
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(1000)
	fmt.Println("Peer client is up")
	ip := "127.0.0.1"
	peer_debug = false
	peer_logger = log.Default()
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-v" || os.Args[i] == "--verbose" {
			peer_debug = true
			fmt.Println("Debug mode is enabled, program log can be found in /log/Peer" + strconv.Itoa(n) + ".log")
		} else if os.Args[i] == "-r" || os.Args[i] == "--remote" {
			var err error
			ip, err = getPublicIP()
			if err != nil {
				log.Fatalln("Impossible to obtain ip address:", err)
			}
		} else if os.Args[i] == "-l" || os.Args[i] == "--local" {
			ip = GetOutboundIP()
		} else {
			fmt.Println("Unknown flag ", os.Args[1])
			return
		}
	}
	if peer_debug {
		var err error
		peer_logger, err = InitLogger("Peer" + strconv.Itoa(n))
		if err != nil {
			log.Fatalln("Logging file cannot be created: ", err)
		}
		peer_logger.Println("Peer client starting in debug mode")
	}
	c.readConf(peer_logger, peer_debug)

	port := c.PeerPort + n
	c.PeerPort = port
	c.PeerIP = ip

	if peer_debug {
		peer_logger.Println("I'll trying to access shared resources")
	}
	client, err := rpc.DialHTTP("tcp", "reg:"+strconv.Itoa(c.RegPort))
	if err != nil {
		if peer_debug {
			peer_logger.Println("Registration service cannot be reached with error: ", err)
		}
		log.Fatalln("Registration service cannot be reached with error: ", err)
	}
	var reply Registration_reply
	myAddress := &Peer{c.PeerIP, c.PeerPort}
	err = client.Call("RegistrationApi.CanIJoin", &myAddress, &reply)
	if err != nil {
		if peer_debug {
			peer_logger.Println("Request to join cannot be send: ", err)
		}
		log.Fatalln("Request to join cannot be send: ", err)
	}
	if reply.Alg == NULL {
		if peer_debug {
			peer_logger.Println("Cannot join mutual exclusion group: too many peers")
		}
		log.Fatalln("To many peers, try again later")
	} else {
		peer_logger.Println("Registered!")
		peer := make([]Peer, len(reply.Peer))
		for i, element := range reply.Peer {
			peer[i] = element
		}

		if reply.Alg == AUTH {
			RicartAgrawala(reply.Index, c, peer, peer_logger, peer_debug)
		} else if reply.Alg == TOKEN {
			Slave(reply.Index, c, peer, peer_logger, peer_debug)
		} else if reply.Alg == QUORUM {
			mask := make([]int, len(reply.Mask))
			for i, element := range reply.Mask {
				mask[i] = element
			}
			Maekawa(reply.Index, c, peer, mask, peer_logger, peer_debug)
		}
	}
}
