package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"os"
	"sync"
	"./websocket"
	"./subscriber"
)

const (
	maxIllegalPacketCounter = 5
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Failed to load an env file: %v\n", err)
		return
	}

	mutex := sync.Mutex{}
	clients := []websocket.WebSocketClient{}
	clientChannel := make(chan websocket.WebSocketClient)

	go func() {
		listener, _ := net.Listen(
			"tcp",
			os.Getenv("CLIENT_SERVER"),
		)
		fmt.Printf("Start client accepting server %v\n", listener.Addr())
		for {
			connection, err := listener.Accept()
			if err != nil {
				fmt.Printf("Failed to listen. retry again.")
				continue
			}

			go func() {
				// Handshake
				wsClient, err := websocket.Upgrade(connection)
				if err != nil {
					fmt.Printf("Disallowed to connect: %v\n", connection.RemoteAddr())
					// Close connection
					_ = connection.Close()
					return
				}
				clientChannel <- *wsClient
			}()
		}
	}()

	go func() {
		for {
			select {
				case client := <-clientChannel:
					fmt.Printf("Client connected %v\n", client.Client.RemoteAddr())
					client.StartListener(
						&clients,
						&mutex,
					)
			}
		}
	}()

	listener, _ := net.Listen(
		"tcp",
		os.Getenv("CAMERA_SERVER"),
	)
	fmt.Printf("Start camera receiving server %v\n", listener.Addr())

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to listen. retry again.")
			continue
		}

		go func() {
			fmt.Printf("[CAMERA] Connected from %v\n", connection.RemoteAddr())
			illegalPacketCounter := maxIllegalPacketCounter
			for {
				if illegalPacketCounter == 0 {
					fmt.Printf("connected from illegal connection.")
					connection.Close()
					return
				}

				data, loops, err := subscriber.SubscribeImageStream(connection)

				if err != nil {
					illegalPacketCounter--
					continue
				}
				illegalPacketCounter = maxIllegalPacketCounter

				websocket.Broadcast(
					&data,
					loops,
					&clients,
					&mutex,
				)

			}
		}()
	}
}