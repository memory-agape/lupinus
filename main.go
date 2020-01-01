package main

import (
	"encoding/binary"
	"fmt"
	"github.com/joho/godotenv"
	"net"
	"os"
	"sync"
	"./websocket"
	"./util"
	"./validator"
)

const (
	CHUNK_SIZE = 8192
	MAX_ILLEGAL_PACKET_COUNTER = 5
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Failed to load an env file: %v\n", err)
		return
	}

	var wg sync.WaitGroup

	clientChannel := make(chan websocket.WebSocketClient)

	wg.Add(1)
	go func() {
		listener, _ := net.Listen(
			"tcp",
			os.Getenv("CLIENT_SERVER"),
		)
		fmt.Printf("Start client accepting server %v\n", listener.Addr())
		for {
			connection, _ := listener.Accept()
			go func() {
				// Handshake
				wsClient, err := websocket.Upgrade(connection)
				if err != nil {
					fmt.Printf("Disallowed to connect: %v\n", connection.RemoteAddr())

					// Close connection
					connection.Close()
					return
				}
				clientChannel <- *wsClient
			}()
		}
	}()

	go func() {
		mutex := sync.Mutex{}
		listener, _ := net.Listen(
			"tcp",
			os.Getenv("CAMERA_SERVER"),
		)

		fmt.Printf("Start camera receiving server %v\n", listener.Addr())

		clients := []websocket.WebSocketClient{}
		go func() {
			for {
				select  {
				case client := <-clientChannel:
					fmt.Printf("Client connected %v\n", client.Client.RemoteAddr())
					mutex.Lock()
					clients = append(clients, client)
					mutex.Unlock()

					go func () {
						for {
							result, opcode, err := client.Decode()
							if err != nil {
								err = client.Client.Close()
								mutex.Lock()
								clients = client.RemoveFromClients(clients)
								mutex.Unlock()
								return
							}

							switch opcode {
							case websocket.OPCODE_CLOSE:
								_, err := client.Client.Write(
									client.Encode(
										result,
										websocket.OPCODE_CLOSE,
										true,
									),
								)
								err = client.Client.Close()
								_ = err

								mutex.Lock()
								clients = client.RemoveFromClients(clients)
								mutex.Unlock()
								return
							case websocket.OPCODE_PING:
								_, err := client.Client.Write(
									client.Encode(
										result,
										websocket.OPCODE_PONG,
										true,
									),
								)
								if err != nil {
									err = client.Client.Close()
									mutex.Lock()
									clients = client.RemoveFromClients(clients)
									mutex.Unlock()
									return
								}
								break
							default:
								// Nothing to do
								break
							}
						}
					}()
				}
			}
		}()

		authKey := os.Getenv("AUTH_KEY")
		authKeySize := len(authKey)

		for {
			connection, _ := listener.Accept()

			go func() {
				fmt.Printf("[CAMERA] Connected from %v\n", connection.RemoteAddr())
				illegalPacketCounter := MAX_ILLEGAL_PACKET_COUNTER
				for {
					if illegalPacketCounter == 0 {
						fmt.Printf("connected from illegal connection.")
						connection.Close()
						return
					}
					readAuthKey := make([]byte, authKeySize)
					receivedAuthKeySize, err := connection.Read(readAuthKey)
					if err != nil {
						fmt.Printf("err = %+v\n", err)

						illegalPacketCounter--
						continue
					}

					// Compare the received auth key and settled auth key.
					if string(readAuthKey[:receivedAuthKeySize]) != authKey {
						fmt.Printf("err = %+v\n", err)

						illegalPacketCounter--
						continue
					}

					// Receive frame size
					frameSize := make([]byte, 4)
					_, errReceivingFrameSize := connection.Read(frameSize)
					if errReceivingFrameSize != nil {
						fmt.Printf("err = %+v\n", err)

						illegalPacketCounter--
						continue
					}

					realFrameSize := binary.BigEndian.Uint32(frameSize)
					realFrame := make([]byte, realFrameSize)

					receivedImageDataSize, errReceivingRealFrame := connection.Read(realFrame)
					if errReceivingRealFrame != nil {
						fmt.Printf("err = %+v\n", err)

						illegalPacketCounter--
						continue
					}

					frameData := realFrame[:receivedImageDataSize]

					if !validator.IsImageJpeg(frameData) {
						illegalPacketCounter--
						continue
					}

					illegalPacketCounter = MAX_ILLEGAL_PACKET_COUNTER

					// Chunk the too long data.
					data, loops := util.Chunk(
						util.Byte2base64URI(
							frameData,
						),
						CHUNK_SIZE,
					)

					for _, client := range clients {
						go func () {
							for i := 0; i < loops; i++ {
								opcode := websocket.OPCODE_BINARY
								if i > 0 {
									opcode = websocket.OPCODE_FIN
								}
								_, err := client.Client.Write(
									client.Encode(
										data[i],
										opcode,
										i == loops,
									),
								)
								if err != nil {
									// Recreate new clients slice.
									fmt.Printf("Failed to write%v\n", client.Client.RemoteAddr())

									mutex.Lock()
									clients = client.RemoveFromClients(clients)
									mutex.Unlock()
								}
							}
						}()
					}
				}
			}()
		}
	}()

	wg.Wait()
}