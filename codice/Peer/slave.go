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

var my_clock Clock
var my_token bool
var Token_logger *log.Logger
var Token_debug bool

func (api *Peer_Api) SendToken(args *bool, reply *int) error {
	my_token = *args
	msg_delay()
	return nil
}

func (api *Peer_Api) ProgMsg(args *Req, reply *int) error {
	for i, element := range my_clock.value {
		if element < (*args).Timestamp[i] {
			my_clock.value[i] = (*args).Timestamp[i]
		}
	}
	msg_delay()
	return nil
}

func Slave(index int, c Conf, peer []Peer, logger *log.Logger, debug bool) {
	Token_logger = logger
	Token_debug = debug
	N := len(peer)
	fmt.Println("Starting...")
	if Token_debug {
		Token_logger.Println("Centralized token algorithm client", index, "started")
	}
	my_clock.New(N)
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

	g.Go(func() error {
		return http.Serve(lis, nil)
	})

	for true {
		my_clock.value[index]++
		args := Req{index, my_clock.value, c.PeerIP, c.PeerPort}
		var reply bool
		client, err := rpc.DialHTTP("tcp", c.MasterIP+":"+strconv.Itoa(c.MasterPort))
		if err != nil {
			if Token_debug {
				Token_logger.Println("Coordinator cannot be reached with error: ", err)
			}
			sigs <- syscall.SIGINT
		}
		msg_delay()
		err = client.Call("API.GetRequest", &args, &reply)
		if err != nil {
			if Token_debug {
				Token_logger.Println("Token request failed with error: ", err)
			}
			sigs <- syscall.SIGINT
		}
		if Token_debug {
			Token_logger.Println("Token request sent")
		}
		for j := 0; j < N; j++ {
			if j != index {
				client, err = rpc.DialHTTP("tcp", peer[j].IP+":"+strconv.Itoa(peer[j].Port))
				if client != nil {
					msg_delay()
					err = client.Call("API.ProgMsg", &args, nil)
					if err != nil {
						if Token_debug {
							Token_logger.Println("Program message failed with error: ", err)
						}
						sigs <- syscall.SIGINT
					}
				}
			}
		}
		my_token = reply
		if my_token {
			CriticSection(Token_logger, Token_debug)
			my_token = false
		} else {
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
		client, err = rpc.DialHTTP("tcp", c.MasterIP+":"+strconv.Itoa(c.MasterPort))
		if err != nil {
			if Token_debug {
				Token_logger.Println("Coordinator cannot be reached with error: ", err)
			}
			sigs <- syscall.SIGINT
		}
		msg_delay()
		err = client.Call("API.ReturnToken", &index, nil)
		if err != nil {
			if Token_debug {
				Token_logger.Println("Token return failed with error: ", err)
			}
			sigs <- syscall.SIGINT
		}

	}
}
