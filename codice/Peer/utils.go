package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
)

type Clock struct {
	len   int
	value []int
}

func (c *Clock) New(n int) {
	(*c).len = n
	(*c).value = make([]int, n)
}
func (c Clock) Min(T []int, index int) bool {
	for i, element := range c.value {
		if index != i && element < T[i] {
			return false
		}
	}
	return true
}

/*type voter struct {
	index int
	vote  bool
}*/

type Quorum struct {
	v     []Peer
	len   int
	enter int
}

func (q *Quorum) Init(index int, n int, peer []Peer, mask []int) {
	k := 0
	for _, element := range mask {
		if element == 1 {
			k++
		}
	}
	(*q).len = k
	(*q).enter = 0
	(*q).v = make([]Peer, k)
	j := 0
	for i := 0; i < len(mask); i++ {
		if mask[i%len(mask)] == 1 {
			q.v[j] = peer[(i+index)%len(peer)]
			j++
		}
	}
}

type State int

const (
	RELEASED = iota
	WANTED
	HELD
)

type Algorithm int

const (
	AUTH = iota
	TOKEN
	QUORUM
	NULL
)

func CriticSection(l *log.Logger, v bool) {
	if v {
		l.Println("Critic section entered")
	}
	fmt.Println("Critic section obtained")
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(100)
	time.Sleep((time.Duration)(n/10) * time.Second)
	if v {
		l.Println("Exiting critic section")
	}

}

type Req struct {
	P         int
	Timestamp []int
	IP        string
	Port      int
}

type Peer struct {
	IP   string
	Port int
}

type Registration_reply struct {
	Peer  []Peer
	Alg   Algorithm
	Index int
	Mask  []int
}

type Conf struct {
	RegPort    int    `json:"reg_port"`
	MasterPort int    `json:"master_port"`
	PeerPort   int    `json:"peer_port"`
	RegIP      string `json:"reg_ip"`
	MasterIP   string `json:"master_ip"`
	PeerIP     string `json:"peer_ip"`
}

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

func findIndex(list []Peer, elem Peer) int {
	for i := range list {
		if list[i] == elem {
			return i
		}
	}
	return -1
}

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

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()

}

func getPublicIP() (string, error) {
	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func msg_delay() {
	d := rand.Intn(2000)
	time.Sleep(time.Duration(d) * time.Millisecond)
}
