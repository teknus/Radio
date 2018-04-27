package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/teknus/Radio/format_msg"
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

func readFromConn(fromControl chan []byte, conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		text, err := reader.ReadBytes(byte('\n'))
		if err == nil {
			fromControl <- text
		}
	}
}

func main() {
	keyBoardInput := make(chan string)
	fromServer := make(chan []byte)
	conn, err := net.Dial("tcp", "localhost:"+os.Args[1])
	udpport, err := strconv.Atoi(os.Args[2])
	udpport16 := uint16(udpport)
	if err != nil {
		return
	}
	go readShell(keyBoardInput)
	go readFromConn(fromServer, conn)
	if err == nil {
		hello := format_msg.PackingMsg(uint8(0), udpport16)
		tempReader := bufio.NewReader(conn)
		conn.Write(hello)
		welcome, err := tempReader.ReadBytes(byte('\n'))
		if err != nil {
			fmt.Println("Handshake error")
		}
		command8, command16 := format_msg.UnpackingMsg(welcome[:len(welcome)-1])
		fmt.Println(command8, command16)
		for {
			select {
			case msg := <-keyBoardInput:
				msg = msg[:len(msg)-1]
				if msg == "q" {
					conn.Close()
					fmt.Println("Client Closed")
					return
				}
				command, err := strconv.Atoi(msg)
				if err == nil {
					station := uint16(command)
					setStation := format_msg.PackingMsg(uint8(1), station)
					conn.Write(setStation)
				} else {
					fmt.Println("station number only")
					continue
				}
			case msg := <-fromServer:
				fmt.Println(msg)
			}
		}
	}
}
