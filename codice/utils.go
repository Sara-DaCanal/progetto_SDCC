package main

import (
	"log"
	"math"
	"math/rand"
	"net/rpc"
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
		if index-1 != i && element < T[i] {
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
)

func CriticSection() {
	log.Println("Critic section entered")
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(100)
	time.Sleep((time.Duration)(n/10) * time.Second)
	log.Println("Exiting critic section")

}

func IWantToRegister(id int) {
	log.Println("I'll trying to access shared resources")
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalln("Registration service cannot be reached with error: ", err)
	}
	port := 8000 + id
	var reply bool
	err = client.Call("RegistrationApi.CanIJoin", &port, &reply)
	if err != nil {
		log.Fatalln("Request to join cannot be send: ", err)
	}
	if reply {
		log.Println("registered")
	}

}
