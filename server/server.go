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
	server.listAll = make(chan bool, 1)
	server.closeAll = make(chan bool, 1)
	server.StartServer(os.Args)
}

type Server struct {
	ln            net.Listener
	changeStation chan *ChangeStation
	toclientsList chan net.Conn
	stations      []*Station
	listAll       chan bool
	closeAll      chan bool
}

func readShell(toControl chan<- string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Shell")
	for {
		text, _ := reader.ReadString('\n')
		toControl <- text
	}
	close(toControl)
}

func (server *Server) startStation(names []string, stationChan []*Station) []*Station {
	stations := make([]*Station, 0)
	for _, name := range names {
		station := &Station{
			buff:      make(chan []byte, 1),
			newClient: make(chan *Client, 1),
			delClient: make(chan *Client, 1),
			name:      name,
			closeAll:  make(chan bool, 1),
			listAll:   make(chan bool, 1),
		}
		go station.musicLoop(name, station.buff)
		go station.HandleClients(station.newClient, station.delClient, station.buff, station.listAll, station.closeAll)
		stations = append(stations, station)
	}
	return stations
}

func (server *Server) StartServer(arg []string) {
	ln, err := net.Listen("tcp", ":"+arg[1])
	if err != nil {
		log.Println(err)
	}
	fromKeyboard := make(chan string, 1)
	server.ln = ln
	temp := make([]string, 0)
	for _, station := range arg[2:] {
		temp = append(temp, station)
	}
	closeServer := make(chan bool, 1)
	go readShell(fromKeyboard)
	go server.CloseLoop(closeServer)
	go server.ReadCommands(fromKeyboard, server.listAll, server.closeAll)
	go server.HandleClients(server.startStation(temp, server.stations), server.changeStation, server.toclientsList, server.listAll, server.closeAll, closeServer)
	server.AcceptConn(server.ln, server.toclientsList)
}

func (server *Server) ReadCommands(fromKeyboard chan string,
	listAll chan bool,
	closeAll chan bool) {
	for {
		select {
		case msg := <-fromKeyboard:
			msg = msg[:len(msg)-1]
			if msg == "p" {
				listAll <- true
			} else if msg == "q" {
				closeAll <- true
			} else {
				fmt.Println("Invalid Command")
			}
		}
	}
}

func (server *Server) AcceptConn(ln net.Listener,
	toclientsList chan net.Conn) {
	fmt.Println("Server Up")
	for {
		conn, err := ln.Accept()
		if err == nil || conn != nil {
			toclientsList <- conn
		}
	}
}

func (server *Server) CloseLoop(closeServer chan bool) {
	for {
		select {
		case <-closeServer:
			server.ln.Close()
			os.Exit(0)
		}
	}
}
func (server *Server) HandleClients(stations []*Station, changeStation chan *ChangeStation,
	toclientsList chan net.Conn,
	listAll chan bool,
	closeAll chan bool,
	closeServer chan bool) {
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
			if int(change.old) < len(stations) {
				stations[int(change.old)].delClient <- change.client
				if int(change.new) < len(stations) && int(change.new) >= 0 {
					change.client.station = change.new
					stations[int(change.new)].newClient <- change.client
				} else {
					msg := "Invalid new Station"
					change.client.send <- format_msg.PackingStringMsg(uint8(2), uint8(len(msg)), msg)
				}
			} else {
				msg := "Invalid Old Station"
				change.client.send <- format_msg.PackingStringMsg(uint8(2), uint8(len(msg)), msg)
			}
		case <-closeAll:
			for _, station := range stations {
				station.closeAll <- true
			}
			closeServer <- true
		case <-listAll:
			for _, stations := range stations {
				stations.listAll <- true
			}
		}
	}
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
				fmt.Println("Fechei closeAll")
				client.conn.Close()
				client.connUDP.Close()
				client.closeKeepAlive <- true
				delete(clientList, key)
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
	conn           net.Conn
	connUDP        net.Conn
	send           chan []byte
	sendStream     chan []byte
	udpPort        string
	station        uint16
	changeStation  chan *ChangeStation
	numStations    uint16
	closeKeepAlive chan bool
}

func (c *Client) HandleConn(conn net.Conn, send chan []byte,
	changestation chan *ChangeStation, sendStream chan []byte) {
	go c.handleMsgs(conn, send, sendStream, changestation)
	for {
		select {
		case buff := <-sendStream:
			if c.connUDP != nil {
				_, err := c.connUDP.Write(buff)
				if err != nil {
					c.closeKeepAlive <- true
					return
				}
			}
		case buff := <-send:
			_, err := conn.Write(buff)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) keepAlive(conn net.Conn, changestation chan *ChangeStation, closeKeepAlive chan bool) {
	for {
		select {
		case <-closeKeepAlive:
			c.conn.Close()
			c.connUDP.Close()
			return
		default:
			time.Sleep(1 * time.Second)
			_, err := conn.Write(format_msg.PackingStringMsg(uint8(3), uint8(0), "Alive"))
			if err != nil {
				c.conn.Close()
				c.connUDP.Close()
				changestation <- &ChangeStation{
					old:    c.station,
					new:    c.numStations,
					client: c,
				}
				break
			}
		}
	}
}

func (c *Client) handleMsgs(conn net.Conn, send chan []byte, sendStream chan []byte,
	changestation chan *ChangeStation) {
	reader := bufio.NewReader(conn)
	c.closeKeepAlive = make(chan bool, 1)
	for {
		command, err := reader.ReadBytes(byte('\n'))
		if err == nil {
			command8, command16 := format_msg.UnpackingMsg(command[:len(command)-1])
			if command8 == uint8(0) {
				c.connUDP, err = net.Dial("udp", "localhost:"+strconv.Itoa(int(command16)))
				if err != nil {
					fmt.Println("Error creating UDP conn")
				}
				send <- format_msg.PackingMsg(uint8(1), c.numStations)
				go c.keepAlive(conn, changestation, c.closeKeepAlive)
			} else if command8 == uint8(1) {
				changestation <- &ChangeStation{
					old:    c.station,
					new:    command16,
					client: c,
				}
			} else if command8 == uint8(2) {
				fmt.Println("Closed Here")
				conn.Close()
				c.connUDP.Close()
				c.closeKeepAlive <- true
				changestation <- &ChangeStation{
					old:    c.station,
					new:    command16,
					client: c,
				}
				break
			}
		} else {
			c.conn.Close()
			c.connUDP.Close()
			changestation <- &ChangeStation{
				old:    c.station,
				new:    c.numStations,
				client: c,
			}
			c.closeKeepAlive <- true
			break
		}
	}
}
