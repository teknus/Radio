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

func handShake(conn net.Conn, udpport uint16) (uint8, uint16) {
	hello := format_msg.PackingMsg(uint8(0), udpport)
	tempReader := bufio.NewReader(conn)
	conn.Write(hello)
	welcome, err := tempReader.ReadBytes(byte('\n'))
	if err != nil {
		fmt.Println("Handshake error")
	}
	return format_msg.UnpackingMsg(welcome)
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
	ui8, ui16 := handShake(conn, udpport16)
	fmt.Println(ui8, ui16)
	go readShell(keyBoardInput)
	go readFromConn(fromServer, conn)
	if err == nil {
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
