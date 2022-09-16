package main

import (
	"container/list"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

var state State
var voted bool
var my_quorum Quorum
var my_index int

type Maekawa_api int

var my_reqList = list.New()

func (api *Maekawa_api) Request(args *int, reply *bool) error {
	log.Println("Request arrived from process ", *args)
	if state == HELD || voted {
		log.Println("process", *args, "in queue")
		my_reqList.PushFront(*args)
		*reply = false
	} else {
		log.Println("I voted for process", *args)
		*reply = true
		voted = true
	}
	return nil
}

func (api *Maekawa_api) Reply(args *int, reply *int) error {
	log.Println("vote arrived from process", *args)
	my_quorum.enter++
	return nil
}

func (api *Maekawa_api) Release(args *int, reply *bool) error {
	log.Println("Process ", *args, "released CS")
	if my_reqList.Len() != 0 {
		e := my_reqList.Front()
		my_reqList.Remove(e)
		item := e.Value.(int)
		if item != my_index {
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item))
			if err != nil {
				log.Fatalln("Process ", item, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", &my_index, nil)
			if err != nil {
				log.Fatalln("Reply cannot be sent to process ", item, " with error: ", err)
			}
			log.Println("I voted for process ", item)
			voted = true
		} else {
			log.Println("I voted for myself")
			my_quorum.enter++
			voted = true
		}
	} else {
		log.Println("I can vote")
		voted = false
	}
	return nil
}

func Maekawa(index int, N int) {
	log.Println("Maekawa algorithm client ", index, " started")
	my_index = index
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
			log.Println("I voted for myself")
			my_quorum.enter++
			voted = true
		}
		for my_quorum.enter < my_quorum.len {
		}
		state = HELD
		CriticSection()
		state = RELEASED
		my_quorum.enter = 0
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
			client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item))
			if err != nil {
				log.Fatalln("Process ", item, " cannot be reached with error: ", err)
			}
			err = client.Call("API.Reply", &my_index, nil)
			if err != nil {
				log.Fatalln("Reply cannot be sent to process ", item, " with error: ", err)
			}
			log.Println("I voted for process ", item)
			voted = true
		} else {
			log.Println("I can vote")
			voted = false
		}
	}
	log.Println("FINISH")
	for true {
	}
}
