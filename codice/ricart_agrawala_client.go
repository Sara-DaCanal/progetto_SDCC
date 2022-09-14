package main

import (
	"container/list"
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

type RA_api int

type Msg struct {
	id    int
	clock int
}

func (api *RA_api) Request(args *Msg, reply *bool) error {
	if state == HELD || (state == WANTED && (last_req < (*args).clock || (last_req == (*&args.clock) && me < (*args).id))) {
		reqQueue.PushFront(*args)
		*reply = false
	} else {
		*reply = true
	}
	if s_clock < (*args).clock {
		s_clock = (*args).clock
	}
	return nil
}

func (api *RA_api) Reply(args *int, reply *int) error {
	replies++
	return nil
}

func RicartAgrawala(index int, n int) {
	me = index
	log.Println("Ricart Agrawala algorithm client ", index, " started")
	my_state = RELEASED
	s_clock = 0
	rpc.RegisterName("API", new(RA_api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":800"+strconv.Itoa(index))
	if e != nil {
		log.Fatalln("Listen failed with error:", e)
	}
	log.Println("Process ", index, " listening on port 800"+strconv.Itoa(index))
	go http.Serve(lis, nil)
	time.Sleep(15 * time.Second)
	log.Println("ready")
	for i := 0; i < 5; i++ {
		log.Println("I want cs")
		my_state = WANTED
		s_clock++
		last_req = s_clock
		m := Msg{index, s_clock}
		var reply bool
		for j := 0; j < n; j++ {
			log.Println("Sending requests")
			if j != index {
				client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(j))
				if err != nil {
					log.Fatalln("Process ", j, " cannot be reached with error: ", err)
				}
				err = client.Call("API.Request", &m, &reply)
				if err != nil {
					log.Fatalln("Request cannot be sent to process ", j, " with error: ", err)
				}
				log.Println("Request sent to process ", j)
				if reply {
					replies++
				}
			}
		}
		for replies < n {
		}
		state = HELD
		CriticSection()
		for e := reqQueue.Front(); e != nil; e.Next() {
			item := e.Value.(Msg)
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item.id))
			if err != nil {
				log.Fatalln("Process ", item.id, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", nil, nil)
			if err != nil {
				log.Fatalln("Reply cannot be sent to process ", item.id, " with error: ", err)
			}
			log.Println("Reply sent to process ", item.id)
		}
		reqQueue = list.New()
		state = RELEASED
		replies = 0
	}
}
