package main

import (
	"container/list"
	"context"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

type Reg_api int

var processList = list.New()
var processNumber int
var exAlgo Algorithm

func (r *Reg_api) CanIJoin(args *int, reply *bool) error {
	if processList.Len() > processNumber {
		*reply = false
	} else {
		processList.PushFront(*args)
		log.Println("Someone is trying to register")
		*reply = true
		if exAlgo == AUTH {
			log.Println("Ricart-Agrawala algorithm")
		} else if exAlgo == TOKEN {
			log.Println("Centralized token algorithm")
		} else if exAlgo == QUORUM {
			log.Println("Maekawa algorithm")
		}
	}
	return nil
}

func register(N int, alg Algorithm) {
	log.Println("I am the registration service")
	processNumber = N
	exAlgo = alg
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Shutdown signal caught, registration service will stop")
		cancel()
	}()

	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalln("Listening failed with error: ", err)
	}
	rpc.RegisterName("RegistrationApi", new(Reg_api))
	rpc.HandleHTTP()
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return http.Serve(lis, nil)
	})
	g.Go(func() error {
		<-gCtx.Done()
		return lis.Close()
	})
	if err := g.Wait(); err != nil {
		log.Println("Registration sevice shutdown")
	}

}
