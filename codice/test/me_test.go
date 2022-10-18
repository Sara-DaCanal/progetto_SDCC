package main

import (
	"bufio"
	"container/list"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type entry struct {
	name  string
	enter time.Time
	exit  time.Time
}

type counter struct {
	name string
	n    int
}

var timeList list.List
var orderedTimeList list.List

func getMin(l *list.List) entry {
	m := (*l).Front()
	min := m.Value.(entry)
	for e := (*l).Front(); e != nil; e = e.Next() {
		elem := e.Value.(entry)
		if elem.enter.Before(min.enter) || (elem.enter.Equal(min.enter) && elem.exit.Before(min.exit)) {
			min = elem
			m = e
		}
	}
	(*l).Remove(m)
	orderedTimeList.PushBack(min)
	return min
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

/* ********************* *
 * Mutual exclusion test *
 * ********************* */
func TestExclusive(t *testing.T) {

	//open log directory
	logFiles, err := os.ReadDir("../logs")
	if err != nil {
		t.Fatal("Can't open log directory:", err)
	}
	layout := "2006/01/02 15:04:05"

	//iterate over log files
	for e := range logFiles {
		item := logFiles[e]

		//open peer logfiles
		if strings.Contains(item.Name(), "Peer") {
			file, err := os.OpenFile("../logs/"+item.Name(), os.O_RDONLY, 0666)
			if err != nil {
				t.Fatal("Can't open file:", err)
			}
			defer file.Close()

			//scan peer logfiles
			scanner := bufio.NewScanner(file)
			var my_elem entry
			for scanner.Scan() {
				count := 0

				//init a new entry for every critic section entrance in log
				if strings.Contains(scanner.Text(), "Critic section entered") {
					if count != 0 {
						//if there are two consecutive entrances in the same file, fails
						t.Fatal("Entering again befor exiting")
					}
					my_elem.name = item.Name()
					timeString := scanner.Text()[:19]
					time_parsed, err := time.Parse(layout, timeString)
					if err != nil {
						t.Fatal("Non parsable string:", err)
					}
					my_elem.enter = time_parsed
					count++
				}

				//close and add new entry when exiting message is found
				if strings.Contains(scanner.Text(), "Exiting critic section") {
					count = 0
					timeString := scanner.Text()[:19]
					time_parsed, err := time.Parse(layout, timeString)
					if err != nil {
						t.Fatal("Non parsable string:", err)
					}
					my_elem.exit = time_parsed
					timeList.PushBack(my_elem)
				}
			}
		}
	}

	//scan list
	for timeList.Len() > 1 {
		min := getMin(&timeList)
		for e := timeList.Front(); e != nil; e = e.Next() {
			item := e.Value.(entry)
			if item.enter.Before(min.exit) && !item.enter.Equal(min.exit) {
				//if there is a new entrance before an exit fails
				t.Fatal("Text failed: process", item.name, "entering in critic section before release from process", min.name)
			}
		}
	}
	//otherwise test successful
	fmt.Println("Critic section has been obtained by only one process at a time")
}

/* ************* *
 * Fairness Test *
 * ************* */
func TestFairness(t *testing.T) {

	//open log dir
	logFiles, err := os.ReadDir("../logs")
	if err != nil {
		t.Fatal("Can't open log directory:", err)
	}
	var length int
	if strings.Compare(logFiles[0].Name(), "Coordinator.log") == 0 {
		length = len(logFiles) - 2
		logFiles = logFiles[1:]
	} else {
		length = len(logFiles) - 1
	}

	//make array with different peers
	names := make([]counter, length)
	for e := range logFiles {
		item := logFiles[e]
		if strings.Contains(item.Name(), "Peer") {
			names[e].name = item.Name()
			names[e].n = 0
		}
	}

	//scan list, for every entry increment corresponding peer counter
	for e := orderedTimeList.Front(); e != nil; e = e.Next() {
		item := e.Value.(entry)
		for n := range names {
			if names[n].name == item.name {
				names[n].n++
			}
		}

		//check for too much difference between counters
		for i := 0; i < len(names)-1; i++ {
			for j := 1; j < len(names); j++ {
				if Abs(names[i].n-names[j].n) > 2 {
					t.Fatal("Not fair: process", names[i].name, "and", names[j].name, "have too much difference")
				}
			}
		}
	}
	fmt.Println("Fairness is granted")
}

/* ************* *
 * Liveness Test *
 * ************* */
func TestLiveness(t *testing.T) {
	var initTime time.Time
	var exitTime time.Time
	count := false
	layout := "2006/01/02 15:04:05"

	//compute different accepted interval based on network congestion
	var seconds time.Duration
	delay := os.Getenv("DELAY")
	switch delay {
	case "fast":
		seconds = 11 * time.Second
		break
	case "medium":
		seconds = 14 * time.Second
		break
	case "slow":
		seconds = 19 * time.Second
		break
	default:
		fmt.Println(delay)
	}

	//open log directory
	logFiles, err := os.ReadDir("../logs")
	if err != nil {
		t.Fatal("Can't open log directory:", err)
	}

	//iterate over log
	for e := range logFiles {
		item := logFiles[e]

		//open peer logfiles
		if strings.Contains(item.Name(), "Peer") {
			file, err := os.OpenFile("../logs/"+item.Name(), os.O_RDONLY, 0666)
			if err != nil {
				t.Fatal("Can't open file:", err)
			}
			defer file.Close()

			//scan peer logfiles
			scanner := bufio.NewScanner(file)
			prec := ""
			for scanner.Scan() {

				//search last message before shutting down
				if strings.Contains(scanner.Text(), "Shutdown signal caught") {
					timeString := prec[:19]
					var err error
					initTime, err = time.Parse(layout, timeString)
					if err != nil {
						t.Fatal("Non parsable string:", err)
					}
					timeString = scanner.Text()[:19]
					exitTime, err = time.Parse(layout, timeString)
					if err != nil {
						t.Fatal("Non parsable string:", err)
					}
					count = true
				}
				prec = scanner.Text()
			}
			if count {
				//compute difference between shutting down message and last message
				if exitTime.Sub(initTime) > seconds {
					fmt.Println(exitTime.Sub(initTime), seconds)
					t.Fatal("Test failed for process", item.Name(), exitTime.Format(layout), initTime.Format(layout))
					break
				}
			}
			if !count {
				t.Fatal("Shutting down message was never registered for process", item.Name())
			}

		}
	}
	fmt.Println("No dead-lock detected")

}
