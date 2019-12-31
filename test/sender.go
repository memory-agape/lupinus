package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"time"
	"os"
	"github.com/joho/godotenv"
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

	exampleImageBuffer, _ := ioutil.ReadFile(dir + "/test.jpeg")
	if err != nil {
		fmt.Printf("Failed to load an image file: %v\n", err)
		return
	}

	fmt.Println(string(exampleImageBuffer))

	for {
		connection, err := net.Dial("tcp", "localhost:31000")
		if err != nil {
			fmt.Println("Retry to connect for the testing")
			time.Sleep(5 * time.Second)
			continue
		}

		counter := 1
		for {
			fmt.Printf("Write a data %d\n", counter)

			buffer := []byte{}
			buffer = append(buffer, os.Getenv("AUTH_KEY")...)
			binary.BigEndian.PutUint32(buffer, uint32(32))


			connection.Write(buffer)
			counter++
			time.Sleep(5 * time.Second)
		}
	}
}