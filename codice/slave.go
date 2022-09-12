package main

import (
	"fmt"
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

func Slave(index int, N int) {
	fmt.Print("Sono lo schiavo ")
	fmt.Println(index)
	rpc.RegisterName("API", new(Slave_api))
	rpc.HandleHTTP()
	lis, _ := net.Listen("tcp", ":800"+strconv.Itoa(index))
	go http.Serve(lis, nil)
	clock = make([]int, N)
	client, _ := rpc.DialHTTP("tcp", "127.0.0.1:8000")
	for i := 0; i < 5; i++ {
		clock[index-1]++
		args := Req{index, clock}
		var reply bool
		client.Call("API.GetRequest", &args, &reply)
		my_token = reply
		if my_token {
			fmt.Println("Sono in sezione critica")
			time.Sleep(5 * time.Second)
			my_token = false
		} else {
			fmt.Println("Sono in attesa")
			for !my_token {

			}
			fmt.Println("Sono in sezione critica")
			time.Sleep(5 * time.Second)
			my_token = false
		}
		fmt.Println("rimando indietro il token")
		reply = true
		client.Call("API.ReturnToken", &reply, nil)

	}
}
