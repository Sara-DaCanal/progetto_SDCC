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
	"time"

	"golang.org/x/sync/errgroup"
)

type Reg_api int

var processList []int
var processNumber int
var current int
var exAlgo Algorithm
var reg_debug bool
var reg_logger *log.Logger

func getParam() {
	if reg_debug {
		reg_logger.Println("Starting new registration group")
	}
	fmt.Print("Insert the number for next group (should be at least 2): ")
	fmt.Scanf("%d", &processNumber)
	for processNumber < 2 {
		fmt.Print("Not enough processes, try again: ")
		fmt.Scanf("%d", &processNumber)
	}
	processList = make([]int, processNumber)
	current = 0
	fmt.Println("Insert which algorithm do you wish to use: ")
	fmt.Println("0 - Ricart Agrawala's algorithm")
	fmt.Println("1 - Centralized token algorithm")
	fmt.Println("2 - Maekawa's algorithm")
	var app int
	fmt.Scanf("%d", &app)
	for app > 2 {
		fmt.Print("Invalid algorithm, try again: ")
		fmt.Scanf("%d", &app)
	}
	exAlgo = Algorithm(app)
}

func sendReply(args int, reply *Registration_reply) {
	processList[current] = args
	(*reply).Alg = exAlgo
	(*reply).Index = current
	current++
}
func (r *Reg_api) CanIJoin(args *int, reply *Registration_reply) error {
	if reg_debug {
		reg_logger.Println("Someone is trying to register")
	}
	if current >= processNumber {
		(*reply).Alg = NULL
		if reg_debug {
			reg_logger.Println("permission denied, too many process")
		}
	} else if current < processNumber-1 {
		sendReply(*args, reply)
		for current < processNumber {
			time.Sleep(time.Microsecond)
		}
		(*reply).Peer = processList
		if exAlgo == QUORUM {
			(*reply).Mask = Qgen(processNumber)
		} else {
			(*reply).Mask = nil
		}
	} else {
		sendReply(*args, reply)
		(*reply).Peer = processList

		if exAlgo == QUORUM {
			(*reply).Mask = Qgen(processNumber)
		} else {
			(*reply).Mask = nil
		}
		if reg_debug {
			reg_logger.Println("Mutual exclusion group completed")
		}
		if exAlgo == TOKEN {
			if reg_debug {
				reg_logger.Println("Master process for centralized token algorithm started")
			}
			go Master(processNumber, reg_debug)
		}

		go func() {
			getParam()
			if reg_debug {
				reg_logger.Println("Registration starting again with ", processNumber, "processes")
			}
			fmt.Println("Registration is starting...")
		}()
	}
	return nil
}

func main() {
	fmt.Println("Registration service is up")
	if len(os.Args) > 1 {
		if os.Args[1] == "-v" || os.Args[1] == "-verbose" {
			reg_debug = true
			fmt.Println("Debug mode is enabled, program log can be found in /log/Registration.log")
		} else {
			fmt.Println("Unknown flag ", os.Args[1])
			return
		}
	} else {
		reg_debug = false
	}
	if reg_debug {
		var err error
		reg_logger, err = InitLogger("Registration")
		if err != nil {
			log.Fatalln("Logging file cannot be created: ", err)
		}
		reg_logger.Println("Registration service starting in debug mode")
	}
	getParam()
	var c Conf
	c.readConf(reg_logger, reg_debug)

	fmt.Println("Registration is starting...")
	current = 0
	processList = make([]int, processNumber)
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if reg_debug {
			reg_logger.Println("Shutdown signal caught, registration service will stop")
		}
		cancel()
	}()

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(c.RegPort))
	if err != nil {
		if reg_debug {
			reg_logger.Println("Listening failed with error: ", err)
		}
		log.Fatalln("Listening failed with error: ", err)

	}
	if reg_debug {
		reg_logger.Println("Registration service listening on port", c.RegPort)
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
		fmt.Println("\nRegistration service shutdown")
	}

}
