package main

import (
	"fmt"
	"net/rpc"
	"time"
)

var clock []int
var my_token bool

func Slave(index int, N int) {
	fmt.Print("Sono lo schiavo ")
	fmt.Println(index)
	clock = make([]int, N)
	client, _ := rpc.DialHTTP("tcp", "127.0.0.1:8000")
	args := Req{index, clock}
	clock[index-1]++
	var reply bool
	client.Call("API.GetRequest", &args, &reply)
	my_token = reply
	if my_token {
		fmt.Println("Sono in sezione critica")
		time.Sleep(10)
	}

	client.Call("API.ReturnToken", *&my_token, nil)
}
