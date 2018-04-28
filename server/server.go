package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/teknus/Radio/format_msg"
)

const buffSize = 1024 * 16

func main() {
	server := &Server{}
	server.StartServer(os.Args)
}

type Server struct {
	ln            net.Listener
	changeStation chan ChangeStation
	toclientsList chan net.Conn
	stations      []*Station
}

func (server *Server) StartServer(arg []string) {
	ln, err := net.Listen("tcp", ":"+arg[1])
	if err != nil {
		log.Println(err)
	}
	server.ln = ln
	server.toclientsList = make(chan net.Conn)
	go server.HandleClients(server.stations, server.changeStation, server.toclientsList)
	server.AcceptConn(server.ln, server.toclientsList)
}

func (server *Server) AcceptConn(ln net.Listener, toclientsList chan net.Conn) {
	fmt.Println("Server Up")
	for {
		conn, err := ln.Accept()
		if err == nil || conn != nil {
			toclientsList <- conn
		}
	}
}

func (server *Server) HandleClients(stations []*Station, changeStation chan ChangeStation,
	toclientsList chan net.Conn) {
	for {
		select {
		case newConn := <-toclientsList:
			client := &Client{conn: newConn}
			go client.ListenConn(client.conn, client.send, client.connUDP)
			fmt.Println("Newclinte", client.udpPort)
		case change := <-changeStation:
			stations[change.old].delClient <- change.client
			stations[change.new].newClient <- change.client
		default:
			continue
		}
	}
}

type Station struct {
	clientList []*Client
	newClient  chan *Client
	delClient  chan *Client
	buff       chan []byte
}

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
					fmt.Println("Restart")
					break
				}
			}
			time.Sleep(1 * time.Second)
			buff <- data
		}
	}
}

func (station *Station) broadcast(toAll chan []byte, clients chan []*Client) {
	for {
		select {
		case buffer := <-toAll:
			clientList := <-clients
			for _, client := range clientList {
				client.send <- buffer
			}
			clients <- clientList
		}
	}
}

func (station *Station) removeClient(delClient chan *Client, clients chan []*Client) {
	for {
		select {
		case client := <-delClient:
			clientList := <-clients
			temp := make([]*Client, 0)
			for _, c := range clientList {
				if c != client {
					temp = append(temp, c)
				}
			}
			clientList = temp
			clients <- clientList
		}
	}
}

func (station *Station) addClient(newClient chan *Client, clients chan []*Client) {
	for {
		select {
		case client := <-newClient:
			clientList := <-clients
			clientList = append(clientList, client)
			clients <- clientList
		}
	}
}

type ChangeStation struct {
	old    uint16
	new    uint16
	client *Client
}

type Client struct {
	conn          net.Conn
	connUDP       net.Conn
	send          chan []byte
	udpPort       string
	station       uint16
	changeStation chan ChangeStation
}

func (c *Client) ListenConn(conn net.Conn, send chan []byte, connUDP net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		select {
		case buff := <-send:
			connUDP.Write(buff)
		default:
			command, err := reader.ReadBytes(byte('\n'))
			if err == nil {
				command8, command16 := format_msg.UnpackingMsg(command[:len(command)-1])
				fmt.Println(command8, command16)
				send <- command
				if err != nil {
					fmt.Println("No client connection")
				}
			}
		}
	}
}
