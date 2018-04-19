package main

import (
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	f, err := os.Open("output.mp3")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	data := make([]byte, 1024*16)
	for {
		_, err = f.Read(data)
		charData := string(data) + "\n" //teste para o delimitador no socket
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			return
		}
		fmt.Println(charData[:len(charData)-2])
		time.Sleep(3)
	}
}
