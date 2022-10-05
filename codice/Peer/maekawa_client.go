/* ************************** *
 * Maekawa algorithm for peer *
 * ************************** */
package main

import (
	"container/list"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

//global variables
var state State             //state: either held, wanted or released
var voted bool              //vote avaiable to the peer
var my_quorum Quorum        //quorum of the peer
var my_index int            //peer's index
var my_peer []Peer          //other peers list
var M_logger *log.Logger    //logger
var M_debug bool            //verbose flag
var inquire_sent bool       //flag to memorize wheter the inquire for a particular request has been sent
var failed int              //flag to memorize how many failed response have been received for a request
var locking_req Maekawa_req //current locking request
var seq_num int             //sequence number for next request
var my_reqList = list.New() //list of pending requests

/* ******************************* *
 * Init varible again upon release *
 * ******************************* */
func releaseVariable() {
	voted = false
	inquire_sent = false
	locking_req = Maekawa_req{1000, 1000}
}

/* ********************************* *
 * Api to receive an inquire request *
 * ********************************* */
func (api *Peer_Api) Inquire(args *int, reply *bool) error {

	//process an inquire request
	*reply = false
	if M_debug {
		M_logger.Println("inquire request arrived by process", *args)
	}
	//wait for all reply
	for state == WANTED && my_quorum.reply < my_quorum.len {
		time.Sleep(time.Millisecond)
	}

	//if a failed reply arrived
	if failed != 0 {

		my_quorum.enter--
		if M_debug {
			M_logger.Println("I should relinquish vote, I have ", my_quorum.enter)
		}
		*reply = true
	} else {
		//if nothing failed
		*reply = false
	}

	return nil
}

/* ************************ *
 * Send a request to quorum *
 * ************************ */
func sendRequest(req Maekawa_req) {
	state = WANTED
	//write reply variable
	var reply bool

	//send request to all process in quorum
	for j := 1; j < my_quorum.len; j++ {
		process := findIndex(my_peer, my_quorum.v[j])

		client, err := rpc.DialHTTP("tcp", my_quorum.v[j].IP+":"+strconv.Itoa(my_quorum.v[j].Port))
		for err != nil && strings.Contains(err.Error(), "connection refused") {
			client, err = rpc.DialHTTP("tcp", my_quorum.v[j].IP+":"+strconv.Itoa(my_quorum.v[j].Port))
		}
		if err != nil && !strings.Contains(err.Error(), "connection refused") {
			if M_debug {
				M_logger.Println("Process ", process, " cannot be reached with error: ", err)
			}
			log.Fatalln("Process ", process, " cannot be reached with error: ", err)
		}
		msg_delay()
		err = client.Call("API.Request_m", &req, &reply)
		if err != nil {
			if M_debug {
				M_logger.Println("Request cannot be sent to process ", process, " with error: ", err)
			}
			log.Fatalln("Request cannot be sent to process ", process, " with error: ", err)
		}
		if M_debug {
			M_logger.Println("Request sent to process ", process)
		}
		my_quorum.reply++ //when the call completes, add one to number of processed requests
		if reply {
			my_quorum.enter++ //if reply is true, add one to the quorum
			if M_debug {
				M_logger.Println("Vote arrived directly by process", process)
			}
		} else {
			failed++ //someone denied access
		}

	}
}

/* ************************ *
 * Api to receive a request *
 * ************************ */
func (api *Peer_Api) Request_m(args *Maekawa_req, reply *bool) error {
	*reply = false //init as false, turn to true if necessary
	//process request
	if M_debug {
		M_logger.Println("Request arrived from process", args.P)
	}

	//update sequence number
	if args.Sequence_n > seq_num {
		seq_num = (*args).Sequence_n
	}

	//if vote is not available
	if voted {
		if M_debug {
			M_logger.Println("Vote is not available at the moment")
		}
		//check if is necessary to send an inquire
		if (*args).isSmallest(my_reqList, locking_req) && !inquire_sent {
			if M_debug {
				M_logger.Println("Inquire request should be sent to process", locking_req.P)
			}
			inquire_sent = true
			var inquire_reply bool //initialize reply variable for inquire
			if locking_req.P == my_index {
				api.Inquire(&my_index, &inquire_reply)
			} else {
				client, err := rpc.DialHTTP("tcp", my_peer[locking_req.P].IP+":"+strconv.Itoa(my_peer[locking_req.P].Port))
				if err != nil {
					if M_debug {
						M_logger.Println("Locking process cannot be reached with error:", err)
					}
					log.Fatalln("Locking process cannot be reached with error:", err)
				}

				msg_delay()
				err = client.Call("API.Inquire", &my_index, &inquire_reply)
				if err != nil {
					if M_debug {
						M_logger.Println("Inquire to process", locking_req.P, "cannot be sent with error:", err)
					}
					log.Fatalln("Inquire to process", locking_req.P, "cannot be sent with error:", err)
				}
			}

			//process inquire reply
			if inquire_reply {
				if M_debug {
					M_logger.Println("Process", locking_req.P, "relinquished is vote, voting again")
				}
				//if relinquish message

				//change state variables

				//put both request in queue
				my_reqList.PushBack(*args)
				my_reqList.PushBack(locking_req)
				releaseVariable()

				//remove a request from the queue
				e := nextRequest(*my_reqList)
				my_reqList.Remove(e)
				item := e.Value.(Maekawa_req)
				//if the reply should be sent to requesting process
				voted = true
				if item == *args {
					if M_debug {
						M_logger.Println("Vote sent to requesting process")
					}
					locking_req.P = (*args).P
					locking_req.Sequence_n = (*args).Sequence_n
					*reply = true
				} else if item.P == my_index {
					//if the reply is sent to the process itself
					if M_debug {
						M_logger.Println("Vote sent to myself")
					}
					locking_req.P = item.P
					locking_req.Sequence_n = item.Sequence_n
					//process a reply
					if M_debug {
						M_logger.Println("vote arrived from process", (*args).P)
					}
					my_quorum.enter++
					failed--
				} else {
					if M_debug {
						M_logger.Println("process", (*args).P, "in queue")
					}

					//send reply to process
					client, err := rpc.DialHTTP("tcp", my_peer[item.P].IP+":"+strconv.Itoa(my_peer[item.P].Port))
					if err != nil {
						if M_debug {
							M_logger.Println("Process", item.P, "cannot be reached with error:", err)
						}
						log.Fatalln("Process", item.P, "cannot be reached with error:", err)
					}
					msg_delay()
					err = client.Call("API.Reply_m", &my_index, nil)
					if err != nil {
						if M_debug {
							M_logger.Println("Reply cannot be sent to process", item.P, "with error:", err)
						}
						log.Fatalln("Reply cannot be sent to process", item.P, "with error:", err)
					}
					if M_debug {
						M_logger.Println("Vote sent to process", item.P)
					}
					locking_req.P = item.P
					locking_req.Sequence_n = item.Sequence_n
				}

			} else {
				//if not relinquish message received
				my_reqList.PushBack(*args)
				if M_debug {
					M_logger.Println("process", (*args).P, "in queue")
				}
			}
		} else {
			//if relinquish message is not necessary
			my_reqList.PushBack(*args)
			if M_debug {
				M_logger.Println("process", (*args).P, "in queue")
			}
		}
	} else {
		//if vote is available
		if M_debug {
			M_logger.Println("vote was available and was granted to requesting process")
		}
		voted = true
		locking_req.P = (*args).P
		locking_req.Sequence_n = (*args).Sequence_n
		*reply = true
	}
	msg_delay()
	return nil
}

/* ********************** *
 * Api to receive a reply *
 * ********************** */
func (api *Peer_Api) Reply_m(args *int, reply *int) error {
	if M_debug {
		M_logger.Println("vote arrived from process", *args)
	}
	my_quorum.enter++
	failed--
	msg_delay()
	return nil
}

/* ************************ *
 * Api to receive a release *
 * ************************ */
func (api *Peer_Api) Release(args *int, reply *bool) error {

	//process a release
	if M_debug {
		M_logger.Println("Process", *args, "released CS")
	}
	releaseVariable()
	//if a new reply can be sent
	if my_reqList.Len() != 0 {
		e := nextRequest(*my_reqList)
		my_reqList.Remove(e)
		item := e.Value.(Maekawa_req)
		//if reply to another process
		if item.P != my_index {
			client, err := rpc.DialHTTP("tcp", my_peer[item.P].IP+":"+strconv.Itoa(my_peer[item.P].Port))
			if err != nil {
				if M_debug {
					M_logger.Println("Process", item.P, "cannot be reached with error:", err)
				}
				log.Fatalln("Process", item.P, "cannot be reached with error:", err)
			}
			msg_delay()
			err = client.Call("API.Reply_m", &my_index, nil)
			if err != nil {
				if M_debug {
					M_logger.Println("Reply cannot be sent to process", item.P, "with error:", err)
				}
				log.Fatalln("Reply cannot be sent to process", item.P, "with error:", err)
			}
			locking_req.P = item.P
			locking_req.Sequence_n = item.Sequence_n
			voted = true
			if M_debug {
				M_logger.Println("Vote sent to process", item.P)
			}

		} else {
			//if reply sent to myself
			if M_debug {
				M_logger.Println("Process voted for itself")
			}
			locking_req.P = item.P
			locking_req.Sequence_n = item.Sequence_n
			voted = true
			my_quorum.enter++
		}
	} else {
		//no reply can be sent
		if M_debug {
			M_logger.Println("Available to vote")
		}
	}
	msg_delay()
	return nil
}

func Maekawa(index int, c Conf, peer []Peer, mask []int, logger *log.Logger, debug bool) {
	fmt.Println("Starting...")

	//initialize state variable
	M_logger = logger
	M_debug = debug
	N := len(peer)
	my_peer = peer
	my_index = index
	state = RELEASED
	voted = false
	failed = 0
	inquire_sent = false
	locking_req = Maekawa_req{1000, 1000}
	seq_num = 0
	my_quorum.Init(index, N, peer, mask)
	if M_debug {
		M_logger.Println("Maekawa algorithm client", index, "started")
		M_logger.Println("Quorum for process", index, "is", my_quorum.v)
	}
	rpc.RegisterName("API", new(Peer_Api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":"+strconv.Itoa(c.PeerPort))
	if e != nil {
		if M_debug {
			M_logger.Println("Listen failed with error:", e)
		}
		log.Fatalln("Listen failed with error:", e)
	}
	if M_debug {
		M_logger.Println("Process listening on ip", c.PeerIP, "and port ", c.PeerPort)
	}
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if M_debug {
			M_logger.Println("Shutdown signal caught, peer service will stop")
		}
		cancel()
		lis.Close()
		fmt.Println("Peer", index, "shutdown")
		os.Exit(0)
	}()
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		return http.Serve(lis, nil)
	})

	for true {
		if M_debug {
			M_logger.Println("Asking to enter critic section")
		}
		//send request
		seq_num++
		req := Maekawa_req{my_index, seq_num}

		//process request to itself
		my_quorum.reply++
		if M_debug {
			M_logger.Println("Request arrived from process itself")
		}

		//if vote is not available
		if voted {
			failed++
			if M_debug {
				M_logger.Println("Vote is not available at the moment")
			}
			//it is never necessary to send an inquire
			my_reqList.PushBack(req)
			if M_debug {
				M_logger.Println("process", req.P, "in queue")
			}

		} else {
			//if vote is available
			if M_debug {
				M_logger.Println("Process voted for itself")
			}
			voted = true
			locking_req.P = req.P
			locking_req.Sequence_n = req.Sequence_n
			//process a reply
			my_quorum.enter++
		}
		//send request to others
		sendRequest(req)

		//wait for all reply
		for my_quorum.enter < my_quorum.len {
		}

		//enter critic section
		state = HELD
		CriticSection(M_logger, M_debug)

		//release critic section
		failed = 0
		state = RELEASED
		my_quorum.enter = 0
		my_quorum.reply = 0

		//release to itself
		releaseVariable()

		//send release to quorum
		for j := 1; j < my_quorum.len; j++ {
			process := findIndex(peer, my_quorum.v[j])
			client, err := rpc.DialHTTP("tcp", my_quorum.v[j].IP+":"+strconv.Itoa(my_quorum.v[j].Port))
			if err != nil {
				if M_debug {
					M_logger.Println("Process ", process, " cannot be reached with error: ", err)
				}
				log.Fatalln("Process ", process, " cannot be reached with error: ", err)
			}
			msg_delay()
			err = client.Call("API.Release", &index, nil)
			if err != nil {
				if M_debug {
					M_logger.Println("Release cannot be sent to process ", process, " with error: ", err)
				}
				log.Fatalln("Release cannot be sent to process ", process, " with error: ", err)
			}
			if M_debug {
				M_logger.Println("Release sent to process ", process)
			}
		}

		//check if a new reply can be sent
		if my_reqList.Len() != 0 {

			//get request to which reply is necessary
			e := nextRequest(*my_reqList)
			my_reqList.Remove(e)
			item := e.Value.(Maekawa_req)

			//if reply should be sent to someone
			if item.P != my_index {
				client, err := rpc.DialHTTP("tcp", peer[item.P].IP+":"+strconv.Itoa(peer[item.P].Port))
				if err != nil {
					if M_debug {
						M_logger.Println("Process ", item.P, " cannot be reached with error: ", err)
					}
					log.Fatalln("Process ", item.P, " cannot be reached with error: ", err)
				}
				msg_delay()
				err = client.Call("API.Reply_m", &my_index, nil)
				if err != nil {
					if M_debug {
						M_logger.Println("Reply cannot be sent to process ", item.P, " with error: ", err)
					}
					log.Fatalln("Reply cannot be sent to process ", item.P, " with error: ", err)
				}
				voted = true
				locking_req.P = item.P
				locking_req.Sequence_n = item.Sequence_n
				if M_debug {
					M_logger.Println("Vote sent to process", item.P)
				}
			} else {
				//send reply to myself
				voted = true
				locking_req.P = item.P
				locking_req.Sequence_n = item.Sequence_n
				my_quorum.enter++
				failed--
				if M_debug {
					M_logger.Println("Process voted for itself")
				}
			}
		} else {
			//no reply can be sent
			if M_debug {
				M_logger.Println("Available to vote")
			}
		}
	}
}
