package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.Dial("udp", "127.0.0.1:1234")
	if err != nil {
		fmt.Printf("Some error %v", err)
		return
	}
	data := make([]byte, 16*1024)
	for {
		f, err := os.Open("output.mp3")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		for {
			_, err = f.Read(data)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Restart")
					break
				}
			}
			time.Sleep(1 * time.Second)
			conn.Write(append(data, byte('\n')))
		}
	}
}
