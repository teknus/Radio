package main

import (
	"os"

	"github.com/teknus/Radio/server"
)

func main() {
	radio := &server.Server{}
	radio.StartServer(os.Args)
}
