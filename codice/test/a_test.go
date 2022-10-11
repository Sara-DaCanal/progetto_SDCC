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

func TestExclusive(t *testing.T) {

	logFiles, err := os.ReadDir("../logs")
	if err != nil {
		t.Fatal("Can't open log directory:", err)
	}
	layout := "2006/01/02 15:04:05"
	for e := range logFiles {
		item := logFiles[e]
		if strings.Contains(item.Name(), "Peer") {
			file, err := os.OpenFile("../logs/"+item.Name(), os.O_RDONLY, 0666)
			if err != nil {
				t.Fatal("Can't open file:", err)
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)

			var my_elem entry
			for scanner.Scan() {
				count := 0
				if strings.Contains(scanner.Text(), "Critic section entered") {
					if count != 0 {
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
	for timeList.Len() > 1 {
		min := getMin(&timeList)
		for e := timeList.Front(); e != nil; e = e.Next() {
			item := e.Value.(entry)
			if item.enter.Before(min.exit) && !item.enter.Equal(min.exit) {
				t.Fatal("Text failed: process", item.name, "entering in critic section before release from process", min.name)
			}
		}
	}
	fmt.Println("Critic section has been obtained by only one process at a time")
}

func TestFairness(t *testing.T) {
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
	names := make([]counter, length)
	for e := range logFiles {
		item := logFiles[e]
		if strings.Contains(item.Name(), "Peer") {
			names[e].name = item.Name()
			names[e].n = 0
		}
	}
	for e := orderedTimeList.Front(); e != nil; e = e.Next() {
		item := e.Value.(entry)
		for n := range names {
			if names[n].name == item.name {
				names[n].n++
			}
		}
		for i := 0; i < len(names)-1; i++ {
			for j := 1; j < len(names); j++ {
				if Abs(names[i].n-names[j].n) > 2 {
					t.Fatal("Not fair")
				}
			}
		}
	}
	fmt.Println("Fairness is granted")
}