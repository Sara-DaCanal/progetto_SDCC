package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strconv"
)

func main() {

	if len(os.Args) != 2 {
		fmt.Println("The sintax is prog_name process_num")
		return
	}
	port, _ := strconv.Atoi(os.Args[1]) //config file

	log.Println("I'll trying to access shared resources")
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatalln("Registration service cannot be reached with error: ", err)
	}
	var reply Registration_reply
	err = client.Call("RegistrationApi.CanIJoin", &port, &reply)
	if err != nil {
		log.Fatalln("Request to join cannot be send: ", err)
	}
	if reply.Alg == NULL {
		log.Fatalln("To many peers, try again later")
	} else {
		log.Println("Registered!")
		if reply.Alg == AUTH {
			RicartAgrawala(reply.Index, reply.N)
		} else if reply.Alg == TOKEN {
			if reply.Index == 0 {
				Master(reply.N)
			} else {
				Slave(reply.Index, reply.N)
			}
		} else if reply.Alg == QUORUM {
			Maekawa(reply.Index, reply.N)
		}
	}
}
