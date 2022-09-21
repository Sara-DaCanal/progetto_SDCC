package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
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
	//v   []voter
	len   int
	enter int
}

func (q *Quorum) Init(index int, n int) {
	app := math.Round(math.Sqrt(float64(n)))
	k := int(app)
	(*q).len = k
	(*q).enter = 0
	/*(*q).v = make([]voter, k)
	for i := 0; i < k; i++ {
		(*q).v[i].index = (index + i) % n
		(*q).v[i].vote = false
	}*/
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

type Req struct {
	P         int
	Timestamp []int
	IP        string
	Port      int
}

type Registration_reply struct {
	Peer  []Peer
	Alg   Algorithm
	Index int
	Mask  []int
}

type Peer struct {
	IP   string
	Port int
}

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
