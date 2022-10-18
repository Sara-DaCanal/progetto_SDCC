package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
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

/* ******************************* *
 * Ricart Agrawala request message *
 * ******************************* */
type Msg struct {
	Id    int
	Clock int
	IP    string
	Port  int
}

/* ****************************************** *
 * Struct to save quorum info and its methods *
 * ****************************************** */
type Quorum struct {
	v     []Peer
	len   int
	enter int
	reply int
}

//init quorum method
func (q *Quorum) Init(index int, n int, peer []Peer, mask []int) {
	k := 0
	for _, element := range mask {
		if element == 1 {
			k++
		}
	}
	(*q).len = k
	(*q).enter = 0
	(*q).reply = 0
	(*q).v = make([]Peer, k)
	j := 0
	for i := 0; i < len(mask); i++ {
		if mask[i%len(mask)] == 1 {
			q.v[j] = peer[(i+index)%len(peer)]
			j++
		}
	}
}

/* ************************************ *
 * State data type with possible values *
 * ************************************ */
type State int

const (
	RELEASED = iota
	WANTED
	HELD
)

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

/* ********************************** *
 * Critic section processing function *
 * ********************************** */
func CriticSection(l *log.Logger, v bool) {
	if v {
		l.Println("Critic section entered")
	}
	fmt.Println("Critic section obtained")
	n := rand.Intn(100)
	seconds := n / 10 * int(time.Second)
	time.Sleep(time.Duration(seconds))
	if v {
		l.Println("Exiting critic section")
	}
	fmt.Println("Exiting critic section")

}

/* ***************************************************** *
 * Struct used to send request from token to coordinator *
 * ***************************************************** */
type Req struct {
	P         int
	Timestamp []int
	IP        string
	Port      int
}

/* ************************************** *
 * Maekawa request message and its method *
 * ************************************** */
type Maekawa_req struct {
	P          int
	Sequence_n int
}

//obtain smallest maekawa req upon all
func (r Maekawa_req) isSmallest(l *list.List, locking Maekawa_req) bool {
	if locking.Sequence_n < r.Sequence_n || (locking.Sequence_n == r.Sequence_n && locking.P < r.Sequence_n) {
		return false
	}
	for e := l.Front(); e != nil; e = e.Next() {
		real_item := e.Value.(Maekawa_req)
		if real_item.P != r.P {
			if real_item.Sequence_n < r.Sequence_n || (real_item.Sequence_n == r.Sequence_n && real_item.P < r.Sequence_n) {
				return false
			}
		}
	}
	return true
}

/* ************************************* *
 * Struct used to save peers information *
 * ************************************* */
type Peer struct {
	IP   string
	Port int
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
 * Find index of a peer in list *
 * **************************** */
func findIndex(list []Peer, elem Peer) int {
	for i := range list {
		if list[i] == elem {
			return i
		}
	}
	return -1
}

/* *************************************** *
 * Function to find next request in a list *
 * *************************************** */
func nextRequest(l list.List) *list.Element {
	min := Maekawa_req{10000, 10000}
	var min_elem *list.Element
	for e := l.Front(); e != nil; e = e.Next() {
		item := e.Value.(Maekawa_req)
		if item.Sequence_n < min.Sequence_n || (item.Sequence_n == min.Sequence_n && item.P < min.P) {
			min.P = item.P
			min.Sequence_n = item.Sequence_n
			min_elem = e
		}
	}
	return min_elem
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

/* ************** *
 * Obtain peer ip *
 * ************** */
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()

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
		d = rand.Intn(4700000)
		d = 300000 + d
	}
	time.Sleep(time.Duration(d) * time.Microsecond)
}
