package main

import (
	"log"
	"math"
	"math/rand"
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

func CriticSection() {
	log.Println("Critic section entered")
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(100)
	time.Sleep((time.Duration)(n/10) * time.Second)

}
