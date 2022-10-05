/* *************************************************** *
 * Coordinator service for centralized token algorithm *
 * *************************************************** */
package main

import (
	"container/list"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
)

type Api int

//global variables
var reqList = list.New()      //list of incoming request
var master_debug bool         //verbose flag
var master_logger *log.Logger //logger
var token bool                //token
var N int                     //number of clients
var my_time Clock             //vectorial clock

/* ******************************* *
 * Api for receiving token request *
 * ******************************* */
func (api *Api) GetRequest(args *Req, reply *bool) error {
	*reply = false

	//add request to the queue
	reqList.PushFront(*args)
	if master_debug {
		master_logger.Println("Token requested by process ", (*args).P, " with timestamp: ", (*args).Timestamp)
	}

	//if the coordinator has the token
	if token {

		//search next eligible request
		for e := reqList.Front(); e != nil; e = e.Next() {
			item := e.Value.(Req)
			if my_time.Min(item.Timestamp, item.P) {
				if item.P == (*args).P {
					//if is from the requesting process send an immediate reply
					if master_debug {
						master_logger.Println("Token sent to requesting process")
					}
					my_time.value[(*args).P] = (*args).Timestamp[(*args).P]
					*reply = true
				} else {
					//else send asyncronous reply to eligible process
					new_reply := true
					client, err := rpc.DialHTTP("tcp", item.IP+":"+strconv.Itoa(item.Port))
					if err != nil {
						if master_debug {
							master_logger.Println("Process ", item.P, " cannot be reached with error: ", err)
						}
						log.Fatalln("Process ", item.P, " cannot be reached with error: ", err)
					}
					msg_delay()
					err = client.Call("API.SendToken", &new_reply, nil)
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

/* ************************************************ *
 * Api for receiving the token again from processes *
 * ************************************************ */
func (api *Api) ReturnToken(args *int, reply *int) error {

	//set token to true again
	token = true
	if master_debug {
		master_logger.Println("Token returned to coordinator by process", *args)
	}

	//search for next elibible request if present
	for e := reqList.Front(); e != nil; e = e.Next() {
		item := e.Value.(Req)
		if my_time.Min(item.Timestamp, item.P) {

			//send token asyncronously to process
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

	//check if verbose mode is enabled and init log file if necessary
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

	//read configuration file and init variables
	var c Conf
	c.readConf(master_logger, master_debug)
	token = true
	N = n
	my_time.New(n)

	//open up port for listening and expose api
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
