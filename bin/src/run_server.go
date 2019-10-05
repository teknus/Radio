package main

import (
	"os"

	"github.com/teknus/radio/server"
)

func main() {
	radio := &server.Server{}
	radio.StartServer(os.Args)
}
