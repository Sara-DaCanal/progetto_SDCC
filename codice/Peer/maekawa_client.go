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

var state State
var voted bool
var my_quorum Quorum
var my_index int
var my_peer []Peer
var M_logger *log.Logger
var M_debug bool

type Maekawa_api int

var my_reqList = list.New()

func (api *Maekawa_api) Request(args *int, reply *bool) error {
	if M_debug {
		M_logger.Println("Request arrived from process", *args)
	}
	if state == HELD || voted {
		if M_debug {
			M_logger.Println("process", *args, "in queue")
		}
		my_reqList.PushFront(*args)
		*reply = false
	} else {
		if M_debug {
			M_logger.Println("Vote sent to process", *args)
		}
		*reply = true
		voted = true
	}
	return nil
}

func (api *Maekawa_api) Reply(args *int, reply *int) error {
	if M_debug {
		M_logger.Println("vote arrived from process", *args)
	}
	my_quorum.enter++
	return nil
}

func (api *Maekawa_api) Release(args *int, reply *bool) error {
	if M_debug {
		M_logger.Println("Process", *args, "released CS")
	}
	if my_reqList.Len() != 0 {
		e := my_reqList.Front()
		my_reqList.Remove(e)
		item := e.Value.(int)
		if item != my_index {
			client, err := rpc.DialHTTP("tcp", my_peer[item].IP+":"+strconv.Itoa(my_peer[item].Port))
			if err != nil {
				if M_debug {
					M_logger.Println("Process", item, "cannot be reached with error:", err)
				}
				log.Fatalln("Process", item, "cannot be reached with error:", err)
			}
			err = client.Call("API.Reply", &my_index, nil)
			if err != nil {
				if M_debug {
					M_logger.Println("Reply cannot be sent to process", item, "with error:", err)
				}
				log.Fatalln("Reply cannot be sent to process", item, "with error:", err)
			}
			if M_debug {
				M_logger.Println("Vote sent to process", item)
			}
			voted = true
		} else {
			if M_debug {
				M_logger.Println("Process voted for itself")
			}
			my_quorum.enter++
			voted = true
		}
	} else {
		if M_debug {
			M_logger.Println("Available to vote")
		}
		voted = false
	}
	return nil
}

func Maekawa(index int, c Conf, peer []Peer, mask []int, logger *log.Logger, debug bool) {
	fmt.Println("Starting...")
	M_logger = logger
	M_debug = debug
	if M_debug {
		M_logger.Println("Maekawa algorithm client", index, "started")
	}
	N := len(peer)
	my_peer = peer
	my_index = index
	my_quorum.Init(index, N, peer, mask)
	if M_debug {
		M_logger.Println("Quorum for process", index, "is", my_quorum.v)
	}
	state = RELEASED
	voted = false
	rpc.RegisterName("API", new(Maekawa_api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":"+strconv.Itoa(c.PeerPort))
	if e != nil {
		if M_debug {
			M_logger.Println("Listen failed with error:", e)
		}
		log.Fatalln("Listen failed with error:", e)
	}
	if M_debug {
		M_logger.Println("Process", index, "listening on ip", c.PeerIP, "and port ", c.PeerPort)
	}
	go http.Serve(lis, nil)
	time.Sleep(time.Duration(index) * time.Millisecond)
	for i := 0; i < 5; i++ {
		if M_debug {
			M_logger.Println("Asking to enter critic section")
		}
		var reply bool
		state = WANTED
		for j := 1; j < my_quorum.len; j++ {
			process := findIndex(peer, my_quorum.v[j])
			client, err := rpc.DialHTTP("tcp", my_quorum.v[j].IP+":"+strconv.Itoa(my_quorum.v[j].Port))
			if err != nil {
				if M_debug {
					M_logger.Println("Process ", process, " cannot be reached with error: ", err)
				}
				log.Fatalln("Process ", process, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Request", &index, &reply)
			if err != nil {
				if M_debug {
					M_logger.Println("Request cannot be sent to process ", process, " with error: ", err)
				}
				log.Fatalln("Request cannot be sent to process ", process, " with error: ", err)
			}
			if M_debug {
				M_logger.Println("Request sent to process ", process)
			}
			if reply {
				my_quorum.enter++
			}
		}
		if voted {
			my_reqList.PushFront(index)
		} else {
			if M_debug {
				M_logger.Println("Process voted for itself")
			}
			my_quorum.enter++
			voted = true
		}
		for my_quorum.enter < my_quorum.len {
		}
		state = HELD
		CriticSection(M_logger, M_debug)
		state = RELEASED
		my_quorum.enter = 0
		for j := 1; j < my_quorum.len; j++ {
			process := findIndex(peer, my_quorum.v[j])
			client, err := rpc.DialHTTP("tcp", my_quorum.v[j].IP+":"+strconv.Itoa(my_quorum.v[j].Port))
			if err != nil {
				if M_debug {
					M_logger.Println("Process ", process, " cannot be reached with error: ", err)
				}
				log.Fatalln("Process ", process, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Release", &index, &reply)
			if err != nil {
				if M_debug {
					M_logger.Println("Release cannot be sent to process ", process, " with error: ", err)
				}
				log.Fatalln("Release cannot be sent to process ", process, " with error: ", err)
			}
			if M_debug {
				M_logger.Println("Release sent to process ", process)
			}
		}
		if my_reqList.Len() != 0 {
			e := my_reqList.Front()
			my_reqList.Remove(e)
			item := e.Value.(int)
			client, err := rpc.DialHTTP("tcp", peer[item].IP+":"+strconv.Itoa(peer[item].Port))
			if err != nil {
				if M_debug {
					M_logger.Println("Process ", item, " cannot be reached with error: ", err)
				}
				log.Fatalln("Process ", item, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", &my_index, nil)
			if err != nil {
				if M_debug {
					M_logger.Println("Reply cannot be sent to process ", item, " with error: ", err)
				}
				log.Fatalln("Reply cannot be sent to process ", item, " with error: ", err)
			}
			if M_debug {
				M_logger.Println("Vote sent to process", item)
			}
			voted = true
		} else {
			if M_debug {
				M_logger.Println("Available to vote")
			}
			voted = false
		}
	}
	fmt.Println("FINISH")
	for true {
	}
}
