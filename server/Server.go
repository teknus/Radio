package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/teknus/Radio/format_msg"
)

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

func (server *Server) StartStation(names []string, stationChan []*Station) []*Station {
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
	server.toclientsList = make(chan net.Conn, 1)
	server.changeStation = make(chan *ChangeStation, 1)
	server.listAll = make(chan bool, 1)
	server.closeAll = make(chan bool, 1)
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
	go server.HandleClients(server.StartStation(temp, server.stations), server.changeStation, server.toclientsList, server.listAll, server.closeAll, closeServer)
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
