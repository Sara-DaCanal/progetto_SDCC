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
	Id    int
	Clock int
}

func (api *RA_api) Request(args *Msg, reply *bool) error {
	if my_state == HELD || (my_state == WANTED && (last_req < (*args).Clock || (last_req == (*&args.Clock) && me < (*args).Id))) {
		reqQueue.PushFront(*args)
		log.Println("Process ", (*args).Id, "in queue")
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
	log.Println("Reply arrived from process ", *args)
	replies++
	return nil
}

func RicartAgrawala(index int, n int) {
	me = index
	log.Println("Ricart Agrawala algorithm client ", index, " started")
	IWantToRegister(index)
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
	time.Sleep(10 * time.Second)
	log.Println("ready")
	for i := 0; i < 5; i++ {
		log.Println("I want cs")
		my_state = WANTED
		s_clock++
		last_req = s_clock
		m := Msg{index, s_clock}
		var reply bool
		for j := 0; j < n; j++ {
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
					log.Println("Process", j, "auth")
					replies++
				}
			}
		}
		for replies < n-1 {
		}
		my_state = HELD
		CriticSection()
		for e := reqQueue.Front(); e != nil; e = e.Next() {
			item := e.Value.(Msg)
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item.Id))
			if err != nil {
				log.Fatalln("Process ", item.Id, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", &index, nil)
			if err != nil {
				log.Fatalln("Reply cannot be sent to process ", item.Id, " with error: ", err)
			}
			log.Println("Reply sent to process ", item.Id)
		}
		reqQueue = list.New()
		my_state = RELEASED
		replies = 0
	}
	for true {
	}
}
