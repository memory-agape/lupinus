package main

import (
	"encoding/binary"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"net"
	"os"
	"time"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Cannot resolve current directory.")
		os.Exit(2)
	}

	err = godotenv.Load(dir + "/../.env")
	if err != nil {
		fmt.Printf("Failed to load an env file: %v\n", err)
		return
	}

	exampleImageBuffer, _ := ioutil.ReadFile(dir + "/test.jpg")
	if err != nil {
		fmt.Printf("Failed to load an image file: %v\n", err)
		return
	}

	for {
		connection, err := net.Dial("tcp", "localhost:30000")
		if err != nil {
			fmt.Println("Retry to connect for the testing")
			time.Sleep(5 * time.Second)
			continue
		}

		counter := 1
		for {
			fmt.Printf("Write a data %d\n", counter)
			fmt.Printf("Size: %d\n", len(exampleImageBuffer))

			buffer := []byte{}
			buffer = append(buffer, []byte(os.Getenv("AUTH_KEY"))...)
			frameBuffer := make([]byte, 4)
			binary.LittleEndian.PutUint32(frameBuffer, uint32(len(exampleImageBuffer)))

			buffer = append(buffer, frameBuffer...)
			buffer = append(buffer, exampleImageBuffer...)

			_, err = connection.Write(buffer)
			if err != nil {
				fmt.Printf("Failed to write %d\n", counter)
			}
			counter++
			time.Sleep(1 * time.Second)
		}
	}
}
