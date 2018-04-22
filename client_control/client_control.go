package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func readShell(toControl chan<- string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Shell")
	for {
		text, _ := reader.ReadString('\n')
		toControl <- text
	}
	close(toControl)
}

func readFromConn(fromControl chan string, conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		text, err := reader.ReadString('\n')
		if err == nil {
			fromControl <- text
		}
	}
}

func main() {
	keyBoardInput := make(chan string)
	fromServer := make(chan string)
	conn, err := net.Dial("tcp", "localhost:"+os.Args[1])
	go readShell(keyBoardInput)
	go readFromConn(fromServer, conn)
	if err == nil {
		for {
			select {
			case msg := <-keyBoardInput:
				conn.Write([]byte(msg))
			case msg := <-fromServer:
				fmt.Println(msg)
			}
		}
	}
}
