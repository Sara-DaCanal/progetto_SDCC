package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("The sintax is prog_name process_num")
		return
	}
	m, _ := strconv.Atoi(os.Args[1])
	index, _ := strconv.Atoi(os.Args[2])
	N, _ := strconv.Atoi(os.Args[3])
	if m == 0 {
		Master(N)
	} else {
		Slave(index, N)
	}
}
