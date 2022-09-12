package main

import (
	"container/list"
	"log"
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
	log.Println("Token requested by process ", (*args).P, " with timestamp: ", (*args).Timestamp)
	if token {
		for e := reqList.Front(); e != nil; e = e.Next() {
			item := e.Value.(Req)
			if min(next, item.Timestamp, item.P) {
				if item.P == (*args).P {
					log.Println("Token sent to process ", item.P)
					next[(*args).P-1] = (*args).Timestamp[(*args).P-1]
					*reply = true
				} /*else {
					fmt.Println("Sono nell'else strano")
					next[item.P-1] = item.Timestamp[item.P-1]
					reply1 := true
					client, _ := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item.P))
					client.Call("API.SendToken", &reply1, nil)
				}*/
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
		log.Println("Token returned to coordinator")
		for e := reqList.Front(); e != nil; e = e.Next() {
			item := e.Value.(Req)
			if min(next, item.Timestamp, item.P) {
				next[item.P-1] = item.Timestamp[item.P-1]
				token = false
				reply := true
				client, err := rpc.DialHTTP("tcp", "127.0.0.1:800"+strconv.Itoa(item.P))
				if err != nil {
					log.Fatalln("Process ", item.P, " cannot be reached with error: ", err)
				}
				err = client.Call("API.SendToken", &reply, nil)
				if err != nil {
					log.Fatalln("Token cannot be sent to process ", item.P, " with error: ", err)
				}
				log.Println("Token sent to process ", item.P)
				reqList.Remove(e)
				break
			}
		}
	}
	return nil
}

func Master(n int) {
	log.Println("Centralized token algorithm coordinator started")
	token = true
	N = n
	next = make([]int, N)
	rpc.RegisterName("API", new(Api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":8000")
	if e != nil {
		log.Fatalln("Listen failed with error:", e)
	}
	log.Println("Coordinator listening on port 8000")
	http.Serve(lis, nil)
}
