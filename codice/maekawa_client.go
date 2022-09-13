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

type State int

const (
	RELEASED = iota
	WANTED
	HELD
)

var state State
var voted bool
var my_quorum Quorum

type Maekawa_api int

var my_reqList = list.New()

func (api *Maekawa_api) Request(args *int, reply *bool) error {
	log.Println("Request arrived from process ", *args)
	if state == HELD || voted {
		my_reqList.PushFront(*args)
		*reply = false
	} else {
		*reply = true
		voted = true
	}
	return nil
}

func (api *Maekawa_api) Reply(args *bool, reply *int) error {
	my_quorum.enter++
	return nil
}

func (api *Maekawa_api) Release(args *int, reply *bool) error {
	log.Println("Process ", *args, "released CS")
	if my_reqList.Len() != 0 {
		e := my_reqList.Front()
		my_reqList.Remove(e)
		item := e.Value.(int)
		my_reply := true
		client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item))
		if err != nil {
			log.Fatalln("Process ", item, " cannot be reached with error: ", err)
		}
		err = client.Call("API.Reply", &my_reply, nil)
		if err != nil {
			log.Fatalln("Reply cannot be sent to process ", item, " with error: ", err)
		}
		log.Println("Reply sent to process ", item)
		voted = true
	} else {
		voted = false
	}
	return nil
}

func Maekawa(index int, N int) {
	log.Println("Maekawa algorithm client ", index, " started")
	my_quorum.Init(index, N)
	state = RELEASED
	voted = false
	rpc.RegisterName("API", new(Maekawa_api))
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
		log.Println("Asking to enter critic section")
		var reply bool
		state = WANTED
		for j := 1; j < my_quorum.len; j++ {
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa((index+j)%N))
			if err != nil {
				log.Fatalln("Process ", (index+j)%N, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Request", &index, &reply)
			if err != nil {
				log.Fatalln("Request cannot be sent to process ", (index+j)%N, " with error: ", err)
			}
			log.Println("Request sent to process ", (index+j)%N)
			if reply {
				my_quorum.enter++
			}
		}
		if voted {
			my_reqList.PushFront(index)
		} else {
			my_quorum.enter++
			voted = true
		}
		for my_quorum.enter < my_quorum.len {
		}
		state = HELD
		CriticSection()
		state = RELEASED
		for j := 1; j < my_quorum.len; j++ {
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa((index+j)%N))
			if err != nil {
				log.Fatalln("Process ", (index+j)%N, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Release", &index, &reply)
			if err != nil {
				log.Fatalln("Release cannot be sent to process ", (index+j)%N, " with error: ", err)
			}
			log.Println("Release sent to process ", (index+j)%N)
		}
		if my_reqList.Len() != 0 {
			e := my_reqList.Front()
			my_reqList.Remove(e)
			item := e.Value.(int)
			my_reply := true
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item))
			if err != nil {
				log.Fatalln("Process ", item, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", &my_reply, nil)
			if err != nil {
				log.Fatalln("Reply cannot be sent to process ", item, " with error: ", err)
			}
			log.Println("Reply sent to process ", item)
			voted = true
		} else {
			voted = false
		}
	}
	log.Println("FINISH")
	for true {
	}
}
