package main

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
)

var s_clock int
var my_state State
var replies int
var reqQueue = list.New()
var last_req int
var me int
var RA_peer []int
var RA_logger *log.Logger
var RA_debug bool

type RA_api int

type Msg struct {
	Id    int
	Clock int
	Port  int
}

func (api *RA_api) Request(args *Msg, reply *bool) error {
	if my_state == HELD || (my_state == WANTED && (last_req < (*args).Clock || (last_req == (*&args.Clock) && me < (*args).Id))) {
		reqQueue.PushFront(*args)
		if RA_debug {
			RA_logger.Println("Process ", (*args).Id, "in queue")
		}
		*reply = false
	} else {
		*reply = true
	}
	if s_clock < (*args).Clock {
		s_clock = (*args).Clock
	}
	return nil
}

func (api *RA_api) Reply(args *int, reply *int) error {
	if RA_debug {
		RA_logger.Println("Reply arrived from process ", *args)
	}
	replies++
	return nil
}

func RicartAgrawala(index int, c Conf, peer []int, logger *log.Logger, debug bool) {
	me = index
	RA_peer = peer
	RA_logger = logger
	RA_debug = debug
	n := len(RA_peer)
	fmt.Println("Starting...")
	if RA_debug {
		RA_logger.Println("Ricart Agrawala algorithm client ", index, " started")
	}
	my_state = RELEASED
	s_clock = 0
	rpc.RegisterName("API", new(RA_api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":"+strconv.Itoa(c.PeerPort))
	if e != nil {
		if RA_debug {
			RA_logger.Println("Listen failed with error:", e)
		}
		log.Fatalln("Listen failed with error:", e)
	}
	if RA_debug {
		RA_logger.Println("Process ", index, " listening on port ", c.PeerPort)
	}
	go http.Serve(lis, nil)
	time.Sleep(time.Millisecond) //forse aumentare su sistema vero
	for i := 0; i < 5; i++ {
		if RA_debug {
			RA_logger.Println("Trying to enter critic section")
		}
		my_state = WANTED
		s_clock++
		last_req = s_clock
		m := Msg{index, s_clock, c.PeerPort} //change to include ip too
		var reply bool
		for j := 0; j < n; j++ {
			peer_port := RA_peer[j]
			if peer_port != c.PeerPort {
				client, err := rpc.DialHTTP("tcp", "127.0.0.1:"+strconv.Itoa(peer_port)) //ip shouldn't be hardcoded
				if err != nil {
					if RA_debug {
						RA_logger.Println("Process ", j, " cannot be reached with error: ", err)
					}
					log.Fatalln("Process ", j, " cannot be reached with error: ", err)
				}
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
				if reply {
					if RA_debug {
						RA_logger.Println("Process", j, "auth")
					}
					replies++
				}
			}
		}
		for replies < n-1 {
		}
		my_state = HELD
		CriticSection(RA_logger, RA_debug)
		for e := reqQueue.Front(); e != nil; e = e.Next() {
			item := e.Value.(Msg)
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:"+strconv.Itoa(item.Port)) //ip shouldn't be hardcoded
			if err != nil {
				if RA_debug {
					RA_logger.Println("Process ", item.Id, " cannot be reached with error: ", err)
				}
				log.Fatalln("Process ", item.Id, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", &index, nil)
			if err != nil {
				if RA_debug {
					RA_logger.Println("Reply cannot be sent to process ", item.Id, " with error: ", err)
				}
				log.Fatalln("Reply cannot be sent to process ", item.Id, " with error: ", err)
			}
			if RA_debug {
				RA_logger.Println("Reply sent to process ", item.Id)
			}
		}
		reqQueue = list.New()
		my_state = RELEASED
		replies = 0
	}
	for true {
	}
}
