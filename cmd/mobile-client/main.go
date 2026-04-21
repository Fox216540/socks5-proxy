package main

import (
	"fmt"
	"mobile-client/client"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: ./mobile-client ip:port")
		return
	}

	serverAddr := os.Args[1]
	client.RunForever(serverAddr)
}
