package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
)

var clock []int
var my_token bool

type Slave_api int

func (api *Slave_api) SendToken(args *bool, reply *int) error {
	my_token = *args
	return nil
}

func criticSection() {
	log.Println("Critic section entered")
	time.Sleep(5 * time.Second)
	my_token = false
}

func (api *Slave_api) ProgMsg(args *Req, reply *int) error {
	for i, element := range clock {
		if element < (*args).Timestamp[i] {
			clock[i] = (*args).Timestamp[i]
		}
	}
	return nil
}

func Slave(index int, N int) {
	log.Println("Centralized token algorithm client ", index)
	rpc.RegisterName("API", new(Slave_api))
	rpc.HandleHTTP()
	lis, err := net.Listen("tcp", ":800"+strconv.Itoa(index))
	if err != nil {
		log.Fatalln("Listen failed with error:", err)
	}
	log.Println("Client ", index, " listening on port 800", index)
	go http.Serve(lis, nil)
	clock = make([]int, N)
	var client *rpc.Client
	for i := 0; i < 5; i++ {
		clock[index-1]++
		args := Req{index, clock}
		var reply bool
		client, err = rpc.DialHTTP("tcp", "127.0.0.1:8000")
		if err != nil {
			log.Fatalln("Coordinator cannot be reached with error: ", err)
		}
		err = client.Call("API.GetRequest", &args, &reply)
		if err != nil {
			log.Fatalln("Token request failed with error: ", err)
		}
		log.Println("Token request sent")
		for j := 0; j < N; j++ {
			if j+1 != index {
				client, err = rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(j+1))
				if client != nil {
					err = client.Call("API.ProgMsg", &args, nil)
					if err != nil {
						log.Fatalln("Program message failed with error: ", err)
					}
				}
			}
		}
		my_token = reply
		if my_token {
			criticSection()
		} else {
			log.Println("Waiting to enter critic section")
			for !my_token {
			}
			criticSection()
		}
		log.Println("Leaving critic section")
		reply = true
		client, err = rpc.DialHTTP("tcp", "127.0.0.1:8000")
		if err != nil {
			log.Fatalln("Coordinator cannot be reached with error: ", err)
		}
		err = client.Call("API.ReturnToken", &reply, nil)
		if err != nil {
			log.Fatalln("Token return failed with error: ", err)
		}

	}
}
