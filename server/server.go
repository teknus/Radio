package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/teknus/Radio/format_msg"
)

const buffSize = 1024 * 16

func main() {
	server := &Server{}
	server.toclientsList = make(chan net.Conn, 1)
	server.changeStation = make(chan *ChangeStation, 1)
	server.StartServer(os.Args)
}

type Server struct {
	ln            net.Listener
	changeStation chan *ChangeStation
	toclientsList chan net.Conn
	stations      []*Station
}

func (server *Server) startStation(names []string, stationChan []*Station) []*Station {
	stations := make([]*Station, 0)
	for _, name := range names {
		station := &Station{
			buff:      make(chan []byte, 1),
			newClient: make(chan *Client, 1),
			delClient: make(chan *Client, 1),
			name:      name,
		}
		go station.musicLoop(name, station.buff)
		go station.HandleClients(station.newClient, station.delClient, station.buff)
		stations = append(stations, station)
	}
	return stations
}

func (server *Server) StartServer(arg []string) {
	ln, err := net.Listen("tcp", ":"+arg[1])
	if err != nil {
		log.Println(err)
	}
	server.ln = ln
	temp := make([]string, 0)
	for _, station := range arg[2:] {
		temp = append(temp, station)
	}
	go server.HandleClients(server.startStation(temp, server.stations), server.changeStation, server.toclientsList)
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

func (server *Server) HandleClients(stations []*Station, changeStation chan *ChangeStation,
	toclientsList chan net.Conn) {
	for {
		select {
		case newConn := <-toclientsList:
			client := &Client{
				conn:          newConn,
				send:          make(chan []byte, 1),
				sendStream:    make(chan []byte, 1),
				changeStation: server.changeStation,
				numStations:   uint16(len(stations)),
				station:       0,
			}
			go client.HandleConn(client.conn, client.send, changeStation, client.sendStream)
		case change := <-changeStation:
			fmt.Println(int(change.old), len(stations))
			if int(change.old) < len(stations) {
				stations[int(change.old)].delClient <- change.client
				if int(change.new) < len(stations) && int(change.new) >= 0 {
					stations[int(change.new)].newClient <- change.client
				} else {
					msg := "Invalid new Station"
					change.client.send <- format_msg.PackingStringMsg(uint8(2), uint8(len(msg)), msg)
				}

			} else {
				msg := "Invalid Old Station"
				change.client.send <- format_msg.PackingStringMsg(uint8(2), uint8(len(msg)), msg)
			}
		default:
			continue
		}
	}
}

type Station struct {
	clientList chan []*Client
	newClient  chan *Client
	delClient  chan *Client
	buff       chan []byte
	name       string
}

func (station *Station) musicLoop(name string, buff chan []byte) {
	data := make([]byte, buffSize)
	music := make([][]byte, 0)
	for {
		if len(music) <= 0 {
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
		for n, i := range music {
			time.Sleep(1 * time.Second)
			if n == len(music) {
				break
			}
			buff <- i
		}
	}
}

func (station *Station) HandleClients(newClient chan *Client, delClient chan *Client,
	toAll chan []byte) {
	clientList := make([]*Client, 0)
	for {
		select {
		case client := <-newClient:
			clientList = append(clientList, client)
			client.send <- format_msg.PackingStringMsg(uint8(1), uint8(len(station.name)), station.name)
		case client := <-delClient:
			temp := make([]*Client, 0)
			for _, c := range clientList {
				if c != client {
					temp = append(temp, c)
				}
			}
			clientList = temp
		case buffer := <-toAll:
			for _, client := range clientList {
				client.sendStream <- buffer
			}
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
	sendStream    chan []byte
	udpPort       string
	station       uint16
	changeStation chan *ChangeStation
	numStations   uint16
}

func (c *Client) HandleConn(conn net.Conn, send chan []byte,
	changestation chan *ChangeStation, sendStream chan []byte) {
	go c.handleMsgs(conn, send, sendStream, changestation)
	for {
		select {
		case buff := <-sendStream:
			if c.connUDP != nil {
				c.connUDP.Write(buff)
			}
		case buff := <-send:
			conn.Write(buff)
		default:
			continue
		}
	}
}

func (c *Client) handleMsgs(conn net.Conn, send chan []byte, sendStream chan []byte,
	changestation chan *ChangeStation) {
	reader := bufio.NewReader(conn)
	for {
		command, err := reader.ReadBytes(byte('\n'))
		if err == nil {
			command8, command16 := format_msg.UnpackingMsg(command[:len(command)-1])
			if command8 == uint8(0) {
				fmt.Println(command8, command16)
				c.connUDP, err = net.Dial("udp", "localhost:"+strconv.Itoa(int(command16)))
				if err != nil {
					fmt.Println("Error creating UDP conn")
				}
				send <- format_msg.PackingMsg(uint8(1), c.numStations)
			} else if command8 == uint8(1) {
				changestation <- &ChangeStation{
					old:    c.station,
					new:    command16,
					client: c,
				}
			}
		}
	}
}
