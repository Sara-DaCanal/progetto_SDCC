package main

import (
	"container/list"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

var reqList = list.New()

type Api int

var token bool
var N int
var next []int

type Req struct {
	P         int
	Timestamp []int
}

func min(V []int, T []int, index int) bool {
	for i, element := range V {
		if index-1 != i && element < T[i] {
			return false
		}
	}
	return true
}

func (api *Api) GetRequest(args *Req, reply *bool) error {
	*reply = false
	reqList.PushFront(*args)
	fmt.Print("Richiesta con timestamp ")
	fmt.Print((*args).Timestamp)
	fmt.Print("dal processo")
	fmt.Println(args.P)
	next[(*args).P-1] = (*args).Timestamp[(*args).P-1]
	if token {
		for e := reqList.Front(); e != nil; e = e.Next() {
			item := e.Value.(Req)
			if min(next, item.Timestamp, (*args).P) { //vanno sistemate alcune cose
				*reply = true
				token = false
				reqList.Remove(e)
				break
			}
		}
	}
	return nil
}

func (api *Api) ReturnToken(args *bool, reply *int) error {
	token = *args
	if token {
		fmt.Println("Ho di nuovo il token")
		for e := reqList.Front(); e != nil; e = e.Next() {
			item := e.Value.(Req)
			if min(next, item.Timestamp, item.P) {
				fmt.Println("sto inviando il token di nuovo")
				token = false
				reply := true
				fmt.Println(item.P)
				client, _ := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item.P))
				client.Call("API.SendToken", &reply, nil)
				reqList.Remove(e)
				break
			}
		}
	}
	return nil
}

func Master(n int) {
	token = true
	fmt.Println("Sono il master")
	N = n
	next = make([]int, N)
	rpc.RegisterName("API", new(Api))
	rpc.HandleHTTP()
	lis, _ := net.Listen("tcp", ":8000")
	http.Serve(lis, nil)
}
