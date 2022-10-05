/* ************************************** *
 * Client for centralized token algorithm *
 * ************************************** */
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/sync/errgroup"
)

//global variable
var my_clock Clock           //vectorial clock
var my_token bool            //token
var Token_logger *log.Logger //logger
var Token_debug bool         //verbose flag

/* *********************** *
 * Api for receiving token *
 * *********************** */
func (api *Peer_Api) SendToken(args *bool, reply *int) error {
	my_token = *args
	msg_delay()
	return nil
}

/* ****************************************************** *
 * Api for receiving program message from the other peers *
 * ****************************************************** */
func (api *Peer_Api) ProgMsg(args *Req, reply *int) error {
	//update the clock
	for i, element := range my_clock.value {
		if element < (*args).Timestamp[i] {
			my_clock.value[i] = (*args).Timestamp[i]
		}
	}
	msg_delay()
	return nil
}

func Slave(index int, c Conf, peer []Peer, logger *log.Logger, debug bool) {

	//init global variables
	Token_logger = logger
	Token_debug = debug
	N := len(peer)
	my_clock.New(N)

	fmt.Println("Starting...")
	if Token_debug {
		Token_logger.Println("Centralized token algorithm client", index, "started")
	}

	//set up listening port and expose api
	rpc.RegisterName("API", new(Peer_Api))
	rpc.HandleHTTP()
	lis, e := net.Listen("tcp", ":"+strconv.Itoa(c.PeerPort))
	if e != nil {
		if Token_debug {
			Token_logger.Println("Listen failed with error:", e)
		}
		log.Fatalln("Listen failed with error:", e)
	}
	if Token_debug {
		Token_logger.Println("Process listening on ip", c.PeerIP, "and port ", c.PeerPort)
	}

	//set up signal handler for shutdown
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	g, _ := errgroup.WithContext(ctx)
	go func() {
		<-sigs
		if Token_debug {
			Token_logger.Println("Shutdown signal caught, peer service will stop")
		}

		cancel()
		lis.Close()
		fmt.Println("Peer", index, "shutdown")
		os.Exit(0)

	}()

	//start serving on listening port
	g.Go(func() error {
		return http.Serve(lis, nil)
	})

	for true {
		//ask for token
		my_clock.value[index]++

		//init request e reply param
		args := Req{index, my_clock.value, c.PeerIP, c.PeerPort}
		var reply bool

		//send request to coordinator
		client, err := rpc.DialHTTP("tcp", c.MasterIP+":"+strconv.Itoa(c.MasterPort))
		if err != nil {
			if Token_debug {
				Token_logger.Println("Coordinator cannot be reached with error: ", err)
			}
			log.Fatalln("Coordinator cannot be reached with error: ", err)
		}
		msg_delay()
		err = client.Call("API.GetRequest", &args, &reply)
		if err != nil {
			if Token_debug {
				Token_logger.Println("Token request failed with error: ", err)
			}
			log.Fatalln("Token request failed with error: ", err)
		}
		if Token_debug {
			Token_logger.Println("Token request sent")
		}

		//send last request clock to all peers
		for j := 0; j < N; j++ {
			if j != index {
				client, err = rpc.DialHTTP("tcp", peer[j].IP+":"+strconv.Itoa(peer[j].Port))
				if client != nil {
					msg_delay()
					//if the peer is not ready, instead of trying again the message isn't sent
					err = client.Call("API.ProgMsg", &args, nil)
					if err != nil {
						if Token_debug {
							Token_logger.Println("Program message failed with error: ", err)
						}
						log.Fatalln("Program message failed with error: ", err)
					}
				}
			}
		}
		my_token = reply

		if my_token {
			//if token is immediately received, enter critic section
			CriticSection(Token_logger, Token_debug)
			my_token = false
		} else {
			//else wait for the token to enter
			if Token_debug {
				Token_logger.Println("Waiting to enter critic section")
			}
			for !my_token {
			}
			CriticSection(Token_logger, Token_debug)
			my_token = false
		}
		if Token_debug {
			Token_logger.Println("Leaving critic section")
		}

		//send back token to master upon completion
		client, err = rpc.DialHTTP("tcp", c.MasterIP+":"+strconv.Itoa(c.MasterPort))
		if err != nil {
			if Token_debug {
				Token_logger.Println("Coordinator cannot be reached with error: ", err)
			}
			log.Fatalln("Coordinator cannot be reached with error: ", err)
		}
		msg_delay()
		err = client.Call("API.ReturnToken", &index, nil)
		if err != nil {
			if Token_debug {
				Token_logger.Println("Token return failed with error: ", err)
			}
			log.Fatalln("Token return failed with error: ", err)
		}

	}
}
