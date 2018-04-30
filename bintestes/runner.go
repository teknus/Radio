package main

import (
	"log"
	"os/exec"
)

func main() {
	for {
		cmd := exec.Command("./client_listnner", "| mpg123 -")
		log.Printf("Running command and waiting for it to finish...")
		err := cmd.Run()
		log.Printf("Command finished with error: %v", err)
	}
}
