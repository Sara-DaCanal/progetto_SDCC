/* ************************* *
 * Utils file for registerer *
 * ************************* */
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"
)

/* ************************************** *
 * Vectorial clock struct and its methods *
 * ************************************** */
type Clock struct {
	len   int
	value []int
}

//init function for vectorial clock
func (c *Clock) New(n int) {
	(*c).len = n
	(*c).value = make([]int, n)
	for i := range (*c).value {
		(*c).value[i] = 0
	}
}

//check which is smaller between two clocks
func (c Clock) Min(T []int, index int) bool {
	for i, element := range c.value {
		if index != i && element < T[i] {
			return false
		}
	}
	return true
}

/* **************************************** *
 * Algorithm data type with possible values *
 * **************************************** */
type Algorithm int

const (
	AUTH = iota
	TOKEN
	QUORUM
	NULL
)

/* ***************************************************** *
 * Struct used to send request from token to coordinator *
 * ***************************************************** */
type Req struct {
	P         int
	Timestamp []int
	IP        string
	Port      int
}

/* ********************************************** *
 * Struct used to reply to a registration request *
 * ********************************************** */
type Registration_reply struct {
	Peer  []Peer
	Alg   Algorithm
	Index int
	Mask  []int
}

/* ************************************* *
 * Struct used to save peers information *
 * ************************************* */
type Peer struct {
	IP   string
	Port int
}

/* *************************** *
 * quorum generating functions *
 * *************************** */
func adjust(r int) int {
	switch r % 3 {
	case 0:
		return r + 2
	case 1:
		return r + 1
	case 2:
		return r
	default:
		return -1
	}
}

func partition(s int, r int, quorum []int) {
	size := r - s + 1
	if size > 7 {
		size = adjust(size)
		x := (size + 1) / 3
		for i := x; i <= 2*x-2; i++ {
			quorum[i] = 0
		}
		partition(s, s+x-1, quorum)
		partition(s+2*x-1, r, quorum)
	} else {
		switch size {
		case 4, 5:
			quorum[s+2] = 0
			break
		case 6, 7:
			quorum[s+3] = 0
			quorum[s+4] = 0
			break
		}
	}
}

func Qgen(n int) []int {
	quorum := make([]int, n)
	k_0 := adjust(n/2 + 1)
	for i := 0; i < n; i++ {
		if i < k_0 {
			quorum[i] = 1

		} else {
			quorum[i] = 0
		}
		partition(0, k_0-1, quorum)
	}
	return quorum
}

/* ************************************************* *
 * Struct to save config information and its methods *
 * ************************************************* */
type Conf struct {
	RegPort    int    `json:"reg_port"`
	MasterPort int    `json:"master_port"`
	PeerPort   int    `json:"peer_port"`
	RegIP      string `json:"reg_ip"`
	MasterIP   string `json:"master_ip"`
	PeerIP     string `json:"peer_ip"`
}

//read config from json file
func (c *Conf) readConf(l *log.Logger, v bool) {
	jsonFile, err := os.Open("./config.json")
	if err != nil {
		if v {
			l.Println("Configuration file cannot be open: ", err)
		}
		log.Fatalln("Configuration file cannot be open: ", err)

	}
	defer jsonFile.Close()
	if v {
		l.Println("Configuration file successfully opened")
	}
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		if v {
			l.Println("Some error occurred while reading from config file: ", err)
		}
		log.Fatalln("Some error occurred while reading from config file: ", err)

	}
	err = json.Unmarshal(byteValue, c)
	if err != nil {
		if v {
			l.Println("Configuration file cannot be decoded: ", err)
		}
		log.Fatalln("Configuration file cannot be decoded: ", err)

	}
	if v {
		l.Println("Configuration successfully loaded")
	}
}

/* **************************** *
 * Initialize log file function *
 * **************************** */
func InitLogger(name string) (*log.Logger, error) {
	logFile, err := os.OpenFile(
		fmt.Sprintf("../logs/%v.log", name),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0666,
	)
	if err != nil {
		return nil, err
	}
	my_log := log.New(logFile, "", log.LstdFlags)
	return my_log, nil
}

/* ****************************************** *
 * Simulate net congestion condition function *
 * ****************************************** */
func msg_delay() {
	delay := os.Getenv("DELAY")
	var d int
	switch delay {
	case "fast":
		d = rand.Intn(88000)
		d = d + 2000
		break
	case "medium":
		d = rand.Intn(440000)
		d = 60000 + d
		break
	case "slow":
		d = rand.Intn(9700000)
		d = 300000 + d
	}
	time.Sleep(time.Duration(d) * time.Microsecond)
}
