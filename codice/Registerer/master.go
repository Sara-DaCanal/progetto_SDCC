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
var master_debug bool
var master_logger *log.Logger

type Api int

var token bool
var N int
var my_time Clock

func (api *Api) GetRequest(args *Req, reply *bool) error {
	*reply = false
	reqList.PushFront(*args)
	if master_debug {
		master_logger.Println("Token requested by process ", (*args).P, " with timestamp: ", (*args).Timestamp)
	}
	if token {
		for e := reqList.Front(); e != nil; e = e.Next() {
			item := e.Value.(Req)
			if my_time.Min(item.Timestamp, item.P) {
				if item.P == (*args).P {
					if master_debug {
						master_logger.Println("Token sent to process ", item.P)
					}
					my_time.value[(*args).P] = (*args).Timestamp[(*args).P]
					*reply = true
				} else {
					reply := true
					client, err := rpc.DialHTTP("tcp", item.IP+":"+strconv.Itoa(item.Port))
					if err != nil {
						if master_debug {
							master_logger.Println("Process ", item.P, " cannot be reached with error: ", err)
						}
						log.Fatalln("Process ", item.P, " cannot be reached with error: ", err)
					}
					msg_delay()
					err = client.Call("API.SendToken", &reply, nil)
					if err != nil {
						if master_debug {
							master_logger.Println("Token cannot be sent to process ", item.P, " with error: ", err)
						}
						log.Fatalln("Token cannot be sent to process ", item.P, " with error: ", err)
					}
					if master_debug {
						master_logger.Println("Token sent to process ", item.P)
					}
				}
				token = false
				reqList.Remove(e)
				break
			}
		}
	}
	msg_delay()
	return nil
}

func (api *Api) ReturnToken(args *int, reply *int) error {
	token = true
	if master_debug {
		master_logger.Println("Token returned to coordinator by process", *args)
	}
	for e := reqList.Front(); e != nil; e = e.Next() {
		item := e.Value.(Req)
		if my_time.Min(item.Timestamp, item.P) {
			my_time.value[item.P] = item.Timestamp[item.P]
			token = false
			reply := true
			client, err := rpc.DialHTTP("tcp", item.IP+":"+strconv.Itoa(item.Port))
			if err != nil {
				if master_debug {
					master_logger.Println("Process ", item.P, " cannot be reached with error: ", err)
				}
				log.Fatalln("Process ", item.P, " cannot be reached with error: ", err)
			}
			msg_delay()
			err = client.Call("API.SendToken", &reply, nil)
			if err != nil {
				if master_debug {
					master_logger.Println("Token cannot be sent to process ", item.P, " with error: ", err)
				}
				log.Fatalln("Token cannot be sent to process ", item.P, " with error: ", err)
			}
			if master_debug {
				master_logger.Println("Token sent to process ", item.P)
			}
			reqList.Remove(e)
			break
		}
	}
	msg_delay()
	return nil
}

func Master(n int, debug bool) {
	master_debug = debug
	master_logger = log.Default()
	if master_debug {
		var err error
		master_logger, err = InitLogger("Coordinator")
		if err != nil {
			log.Fatalln("Logging file cannot be created: ", err)
		}
		master_logger.Println("Coordinator service starting in debug mode")
	}
	master_logger.Println("Centralized token algorithm coordinator started")

	var c Conf
	c.readConf(master_logger, master_debug)
	token = true
	N = n
	my_time.New(n)
	rpc.RegisterName("API", new(Api))
	lis, e := net.Listen("tcp", ":"+strconv.Itoa(c.MasterPort))
	if e != nil {
		if master_debug {
			master_logger.Println("Listen failed with error:", e)
		}
		log.Fatalln("Listen failed with error:", e)

	}
	if master_debug {
		master_logger.Println("Coordinator listening on port", c.MasterPort)
	}
	http.Serve(lis, nil)
}
