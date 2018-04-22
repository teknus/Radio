package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const buffSize = 1024 * 16

func loadMusic(name string, stationChan chan []byte) {
	f, err := os.Open(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	data := make([]byte, buffSize)
	for {
		for {
			_, err = f.Read(data)
			if err != nil {
				if err == io.EOF {
					fmt.Printf("Restart Music %s\n", name)
					break
				}
				fmt.Println(err)
				return
			}
			stationChan <- data
		}
	}

}

func playLocal() {
	f, err := os.Open("output.mp3")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	data := make([]byte, buffSize)
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
		time.Sleep(2)
	}
}

func main() {
	server := &Server{}
	server.StartServer(os.Args)
	server.AcceptConn()
}

type Station struct {
	clientList []*Client
	newClient  chan net.Conn
}

type Client struct {
	conn     net.Conn
	udpPort  string
	stations []*Station
}

func (c *Client) ListenConn() {
	reader := bufio.NewReader(c.conn)
	for {
		command, err := reader.ReadString('\n')
		if err == nil {
			c.conn.Write([]byte(command + "@"))
			if err != nil {
				fmt.Println("No client connection")
			}
		}
	}
}

type Server struct {
	ln            net.Listener
	toclientsList chan net.Conn
	stations      []Station
}

func (server *Server) StartServer(arg []string) {
	ln, err := net.Listen("tcp", ":"+arg[1])
	if err != nil {
		log.Println(err)
	}
	server.ln = ln
	server.toclientsList = make(chan net.Conn)
}

func (server *Server) AcceptConn() {
	go server.HandleClients()
	for {
		conn, err := server.ln.Accept()
		if err == nil || conn != nil {
			server.toclientsList <- conn
		}
	}
}

func (server *Server) HandleClients() {
	for {
		select {
		case newConn := <-server.toclientsList:
			client := &Client{conn: newConn}
			go client.ListenConn()
			fmt.Println("Newclinte", client.udpPort)
		default:
			continue
		}
	}
}
