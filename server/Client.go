package server

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/teknus/radio/format_msg"
)

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

func (c *Client) HandleConn(conn net.Conn, sendMsgControl chan []byte,
	changestation chan *ChangeStation, sendStream chan []byte) {
	go c.handleMsgs(conn, sendMsgControl, sendStream, changestation)
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
		case buff := <-sendMsgControl:
			_, err := conn.Write(buff)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) keepAlive(conn net.Conn, outstation chan *ChangeStation, closeKeepAlive chan bool) {
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
				outstation <- &ChangeStation{
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
