package server

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/teknus/radio/format_msg"
)

const buffSize = 1024 * 16

type ChangeStation struct {
	old    uint16
	new    uint16
	client *Client
}

type Station struct {
	clientList map[*Client]Client
	newClient  chan *Client
	delClient  chan *Client
	buff       chan []byte
	name       string
	closeAll   chan bool
	listAll    chan bool
}

const sleep = 1000

func (station *Station) musicLoop(name string, buff chan []byte) {
	data := make([]byte, buffSize)
	for {
		f, err := os.Open(name)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		for {
			_, err = f.Read(data)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Restart ", name)
					break
				}
			}
			time.Sleep(sleep * time.Millisecond)
			buff <- data
		}
	}
}

func (station *Station) HandleClients(newClient chan *Client, delClient chan *Client,
	toAll chan []byte, listAll chan bool, closeAll chan bool) {
	clientList := make(map[*Client]Client)
	station.clientList = clientList
	for {
		select {
		case client := <-newClient:
			clientList[client] = *client
			client.send <- format_msg.PackingStringMsg(uint8(1), uint8(len(station.name)), station.name)
		case client := <-delClient:
			delete(clientList, client)
		case buffer := <-toAll:
			for _, client := range clientList {
				client.sendStream <- buffer
			}
		case <-listAll:
			if len(clientList) == 0 {
				fmt.Println("Station: ", station.name)
				fmt.Println("Empty\n")
			} else {
				for _, client := range clientList {
					fmt.Println("Station: ", station.name, "\nClient Control: ", client.conn.RemoteAddr().String(), "\nClient UDP", client.connUDP.RemoteAddr().String(), "\n")
				}
			}
		case <-closeAll:
			for key, client := range clientList {
				fmt.Println("closeAll")
				client.conn.Close()
				client.connUDP.Close()
				client.closeKeepAlive <- true
				delete(clientList, key)
			}
		}
	}
}
