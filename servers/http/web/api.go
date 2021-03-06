package web

import (
	"context"
	"encoding/json"
	"fmt"
	"lupinus/client"
	"lupinus/servers/http"
	"lupinus/util"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type afterProcessStruct struct {
	body       http.HttpBody
	clientMeta http.HttpClientMeta
}

func Listen(ctx context.Context) {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	listener, _ := net.Listen(
		"tcp",
		os.Getenv("CLIENT_API_SERVER"),
	)
	fmt.Printf("Start client API server %v\n", listener.Addr())
	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to listen. retry again.")
			continue
		}

		go func(childCtx context.Context) {
			var afterProcess *afterProcessStruct
			_, cancel := context.WithCancel(childCtx)
			defer func() {
				connection.Close()
				cancel()
				if afterProcess != nil {
					afterProcess.body.AfterProcess(
						afterProcess.clientMeta,
					)
				}
			}()

			fmt.Printf("Connected from: %v\n", connection.RemoteAddr())

			// Get headers
			headers, _ := client.GetAllHeaders(&connection)
			result, err := client.FindHeaderByKey(headers, "")

			if err != nil {
				return
			}

			// Set cookies
			cookie, noCookie := client.FindHeaderByKey(headers, "cookie")
			cookies := []http.Cookie{}
			if noCookie == nil {
				for _, item := range strings.Split(cookie.Value, ";") {
					pair := strings.Split(item, "=")
					if len(pair) != 2 {
						continue
					}
					cookies = append(
						cookies,
						http.Cookie{
							Name:  strings.TrimSpace(pair[0]),
							Value: strings.TrimSpace(pair[1]),
						},
					)
				}
			}

			status := util.SplitWithFiltered(result.Value, " ")
			if len(status) != 3 {
				return
			}

			method := strings.ToLower(status[0])
			path := status[1]
			protocol := status[2]

			// Next find body size
			result, err = client.FindHeaderByKey(headers, "content-length")

			requestBody := []byte{}
			if result != nil {
				number, _ := strconv.Atoi(result.Value)
				reads := make([]byte, number)
				connection.Read(reads)
				requestBody = append(requestBody, reads...)
			}

			urlObject, parseError := url.Parse(path)

			if parseError != nil {
				fmt.Printf("Invalid header")
				return
			}

			// TODO: Validator

			clientMeta := http.HttpClientMeta{
				Pipe:     &connection,
				Method:   strings.ToUpper(method),
				Path:     *urlObject,
				Protocol: protocol,
				Payload:  requestBody,
				Cookies:  cookies,
				Headers:  headers,
			}

			originHeader, err := client.FindHeaderByKey(clientMeta.Headers, "origin")
			origin := "*"
			if err == nil {
				origin = originHeader.Value
			}

			// For the preflight
			if clientMeta.Method == "OPTIONS" {
				writeData := "" +
					clientMeta.Protocol + " " + util.GetStatusCodeWithNameByCode(200) + "\n" +
					"Content-Length: 0\n" +
					"Connection: close\n" +
					"Access-Control-Allow-Origin: " + origin + "\n" +
					"Access-Control-Allow-Credentials: true\n" +
					"Access-Control-Allow-Method: *\n" +
					"Access-Control-Allow-Headers: content-type, x-auth-key\n" +
					"\n"

				// Set cookies
				for _, cookie := range http.GetCookies() {
					writeData += "Set-Cookie: " + cookie + "\n"
				}
				(*clientMeta.Pipe).Write([]byte(writeData))
				return
			}

			responseBody, responseHeader, _ := Connect(clientMeta)

			// If which is invalid processing, connection to shutdown.
			if responseBody == nil || responseHeader == nil {
				writeData := "" +
					clientMeta.Protocol + " " + util.GetStatusCodeWithNameByCode(404) + "\n" +
					"Content-Length: 0\n" +
					"Connection: close\n" +
					"Access-Control-Allow-Origin: " + origin + "\n" +
					"Access-Control-Allow-Credentials: true\n" +
					// for Preflight request
					"Access-Control-Allow-Method: *\n" +
					"Access-Control-Allow-Headers: content-type, x-auth-key\n" +
					"\n"

				// Set cookies
				for _, cookie := range http.GetCookies() {
					writeData += "Set-Cookie: " + cookie + "\n"
				}
				(*clientMeta.Pipe).Write([]byte(writeData))
				return
			}

			var contentType string
			var body string
			statusWithName := util.GetStatusCodeWithNameByCode(
				responseHeader.Status,
			)

			if responseBody.RawMode == false {
				contentType = "application/json"
				resultJSON, _ := json.Marshal(responseBody.Payload)

				body = string(resultJSON)
			} else {
				contentType = responseHeader.ContentType
				body = responseBody.Payload["body"].(string)
			}

			// Write buffer
			writeData := "" +
				clientMeta.Protocol + " " + statusWithName + "\n" +
				"Content-Length: " + strconv.Itoa(len(body)) + "\n" +
				"Content-Type: " + contentType + "\n" +
				"Connection: close\n" +
				"Access-Control-Allow-Origin: " + origin + "\n" +
				"Access-Control-Allow-Credentials: true\n" +
				// for Preflight request
				"Access-Control-Allow-Method: *\n" +
				"Access-Control-Allow-Headers: content-type, x-auth-key\n" +
				""

			// Set cookies
			for _, cookie := range http.GetCookies() {
				writeData += "Set-Cookie: " + cookie + "\n"
			}

			writeData += "\n" + body

			connection.Write(
				[]byte(writeData),
			)

			// After process
			if responseBody.AfterProcess != nil {
				afterProcess = &afterProcessStruct{
					body:       *responseBody,
					clientMeta: clientMeta,
				}
			}
		}(childCtx)
	}
}
