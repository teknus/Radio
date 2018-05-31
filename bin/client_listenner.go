package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
)

func main() {
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("Error command line")
	}
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("127.0.0.1"),
	}

	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}
	buff := make([]byte, 1024*16)
	reader := bufio.NewReader(ser)
	for {
		_, _ = reader.Read(buff)
		charData := string(buff)
		fmt.Println(charData[:len(charData)-1])
	}
}
