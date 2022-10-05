/* ***************************************** *
 * Registration service for mutual exclusion *
 * ***************************************** */

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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

//global variables
var processList []Peer     //list of ip and port of participants
var processNumber int      //number of process that will be accepted
var current int            //current process accepted
var exAlgo Algorithm       //algorithm to use for mutual exclusion
var reg_debug bool         //verbose flag
var reg_logger *log.Logger //logger

/* ********************* *
 * Set user input params *
 * ********************* */
func getParam() {
	if reg_debug {
		reg_logger.Println("Starting new registration group")
	}
	processNumber, _ = strconv.Atoi(os.Getenv("N"))
	processList = make([]Peer, processNumber)
	current = 0
	app := os.Getenv("ALG")
	switch app {
	case "auth":
		exAlgo = AUTH
		break
	case "token":
		exAlgo = TOKEN
		break
	case "quorum":
		exAlgo = QUORUM
		break
	}
	if reg_debug {
		reg_logger.Println("Registration starting with", processNumber, "processes using algorithm:", exAlgo)
	}
}

/* ***************************************** *
 * Set reply information for joining request *
 * ***************************************** */
func sendReply(args Peer, reply *Registration_reply) {
	processList[current] = args
	(*reply).Alg = exAlgo
	(*reply).Index = current
	current++
}

/* ********************************** *
 * Api for receiving joining requests *
 * ********************************** */
func (r *Reg_api) CanIJoin(args *Peer, reply *Registration_reply) error {
	if reg_debug {
		reg_logger.Println("Someone is trying to register")
	}

	//start coordinator if the required algorithm is centralized
	if exAlgo == TOKEN && current == 0 {
		if reg_debug {
			reg_logger.Println("Master process for centralized token algorithm started")
		}
		go Master(processNumber, reg_debug)
	}
	if current >= processNumber {

		//refuse connection if too many peer are already present
		(*reply).Alg = NULL
		if reg_debug {
			reg_logger.Println("permission denied, too many process")
		}
	} else if current < processNumber-1 {

		//accept connection request
		sendReply(*args, reply)
		for current < processNumber {
			//wait for all enough request before sending a reply
			time.Sleep(time.Microsecond)
		}
		(*reply).Peer = processList

		//generate quorum for maekawa algorithm if necessary
		if exAlgo == QUORUM {
			(*reply).Mask = Qgen(processNumber)
		} else {
			(*reply).Mask = nil
		}
	} else {

		//accept last request and send reply
		sendReply(*args, reply)
		(*reply).Peer = processList

		//generate quorum for maekawa algorithm if necessary
		if exAlgo == QUORUM {
			(*reply).Mask = Qgen(processNumber)
		} else {
			(*reply).Mask = nil
		}
		if reg_debug {
			reg_logger.Println("Mutual exclusion group completed")
		}
	}
	msg_delay()
	return nil
}

func main() {
	fmt.Println("Registration service is up")

	//init random generator for delays
	rand.Seed(time.Now().UnixNano())

	//check if verbose mode is enabled and init log file if necessary
	reg_debug = false
	reg_logger = log.Default()
	if os.Getenv("VERBOSE") == "1" {
		reg_debug = true
		fmt.Println("Debug mode is enabled, program log can be found in /log/Registration.log")
		var err error
		reg_logger, err = InitLogger("Registration")
		if err != nil {
			log.Fatalln("Logging file cannot be created: ", err)
		}
		reg_logger.Println("Registration service starting in debug mode")
	}
	//read user input params
	getParam()

	//read configuration
	var c Conf
	c.readConf(reg_logger, reg_debug)

	//init variables and signal handler for shutdown
	current = 0
	processList = make([]Peer, processNumber)
	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	//cancel context on shutdown
	go func() {
		<-sigs
		if reg_debug {
			reg_logger.Println("Shutdown signal caught, registration service will stop")
		}
		cancel()
	}()

	fmt.Println("Registration is starting...")

	//set up listening for incoming connection
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

	//expose api on open port
	rpc.RegisterName("RegistrationApi", new(Reg_api))
	rpc.HandleHTTP()
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return http.Serve(lis, nil)
	})
	//close listener on shutdown
	g.Go(func() error {
		<-gCtx.Done()
		return lis.Close()
	})
	if err := g.Wait(); err != nil {
		fmt.Println("\nRegistration service shutdown")
	}

}
