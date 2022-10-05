/* ********************************** *
 * Ricart Agrawala algorithm for peer *
 * ********************************** */
package main

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sync/errgroup"
)

//global variables
var s_clock int           //scalar clock
var my_state State        //state: either held, wanted or released
var replies int           //number of received replies
var reqQueue = list.New() //list of pending requests
var last_req int          //clock of last sent request
var me int                //peer's index
var RA_peer []Peer        //other peers list
var RA_logger *log.Logger //logger
var RA_debug bool         //verbose flag

/* *********************************** *
 * Api for receiving incoming requests *
 * *********************************** */
func (api *Peer_Api) Request(args *Msg, reply *bool) error {
	if my_state == HELD || (my_state == WANTED && (last_req < (*args).Clock || (last_req == (*&args.Clock) && me < (*args).Id))) {
		//if the process is in critic section or waiting to enter with a preceding request
		reqQueue.PushFront(*args)
		if RA_debug {
			RA_logger.Println("Process ", (*args).Id, "in queue")
		}
		*reply = false
	} else {
		//if the process can vote for the incoming request
		*reply = true
	}
	//update the clock
	if s_clock < (*args).Clock {
		s_clock = (*args).Clock
	}
	msg_delay()
	return nil
}

/* ************************* *
 * Api for receiving a reply *
 * ************************* */
func (api *Peer_Api) Reply(args *int, reply *int) error {
	if RA_debug {
		RA_logger.Println("Reply arrived from process ", *args)
	}
	replies++
	msg_delay()
	return nil
}

func RicartAgrawala(index int, c Conf, peer []Peer, logger *log.Logger, debug bool) {

	//init global variables
	me = index
	RA_peer = peer
	RA_logger = logger
	RA_debug = debug
	n := len(RA_peer)
	my_state = RELEASED
	s_clock = 0

	fmt.Println("Starting...")
	if RA_debug {
		RA_logger.Println("Ricart Agrawala algorithm client ", index, " started")
	}

	//set up a listening port and expose APIs on the port
	rpc.RegisterName("API", new(Peer_Api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":"+strconv.Itoa(c.PeerPort))
	if e != nil {
		if RA_debug {
			RA_logger.Println("Listen failed with error:", e)
		}
		log.Fatalln("Listen failed with error:", e)
	}
	if RA_debug {
		RA_logger.Println("Process listening on ip", c.PeerIP, "and port ", c.PeerPort)
	}

	//set up signal handler for shutdown
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if RA_debug {
			RA_logger.Println("Shutdown signal caught, peer service will stop")
		}
		cancel()
		lis.Close()
		fmt.Println("Peer", index, "shutdown")
		os.Exit(0)
	}()
	g, _ := errgroup.WithContext(ctx)

	//start the server on the opened port in asyncronous mode
	g.Go(func() error {
		return http.Serve(lis, nil)
	})

	for true {

		//send request to enter critic section
		if RA_debug {
			RA_logger.Println("Trying to enter critic section")
		}

		//update clock and state variables
		my_state = WANTED
		s_clock++
		last_req = s_clock

		//init request and reply var
		m := Msg{index, s_clock, c.PeerIP, c.PeerPort}
		var reply bool

		//send requests to all peers
		for j := 0; j < n; j++ {
			peer_addr := RA_peer[j]
			if peer_addr.Port != c.PeerPort || peer_addr.IP != c.PeerIP {
				client, err := rpc.DialHTTP("tcp", peer_addr.IP+":"+strconv.Itoa(peer_addr.Port))
				for err != nil && strings.Contains(err.Error(), "connection refused") {
					client, err = rpc.DialHTTP("tcp", peer_addr.IP+":"+strconv.Itoa(peer_addr.Port))
				}
				if err != nil && !strings.Contains(err.Error(), "connection refused") {
					if RA_debug {
						RA_logger.Println("Process ", j, " cannot be reached with error: ", err)
					}
					log.Fatalln("Process ", j, " cannot be reached with error: ", err)
				}
				msg_delay()
				err = client.Call("API.Request", &m, &reply)
				if err != nil {
					if RA_debug {
						RA_logger.Println("Request cannot be sent to process ", j, " with error: ", err)
					}
					log.Fatalln("Request cannot be sent to process ", j, " with error: ", err)
				}
				if RA_debug {
					RA_logger.Println("Request sent to process ", j)
				}

				//receive reply
				if reply {
					if RA_debug {
						RA_logger.Println("Process", j, "auth")
					}
					//add to positive reply count if positive
					replies++
				}
			}
		}

		//wait for positive replies from all peers
		for replies < n-1 {
		}

		//enter critic section
		my_state = HELD
		CriticSection(RA_logger, RA_debug)

		//upon exiting from critic section, send reply to all waiting processes
		for e := reqQueue.Front(); e != nil; e = e.Next() {
			item := e.Value.(Msg)
			client, err := rpc.DialHTTP("tcp", item.IP+":"+strconv.Itoa(item.Port))
			if err != nil {
				if RA_debug {
					RA_logger.Println("Process", item.Id, "cannot be reached with error:", err)
				}
				log.Fatalln("Process", item.Id, "cannot be reached with error:", err)
			}
			msg_delay()
			err = client.Call("API.Reply", &index, nil)
			if err != nil {
				if RA_debug {
					RA_logger.Println("Reply cannot be sent to process", item.Id, "with error:", err)
				}
				log.Fatalln("Reply cannot be sent to process", item.Id, "with error:", err)
			}
			if RA_debug {
				RA_logger.Println("Reply sent to processs", item.Id)
			}
		}

		//reset variables
		reqQueue = list.New()
		my_state = RELEASED
		replies = 0
	}
}
