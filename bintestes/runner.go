package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("./client_listenner | mpg123 - ", "")
	fmt.Printf("Running command and waiting for it to finish...\n")
	err := cmd.Run()
	fmt.Printf("Command finished with error: %v\n", err)
}
