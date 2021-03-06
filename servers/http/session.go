package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lupinus/config"
	"lupinus/util"
	"os"
)

var sessionDir = config.GetRootDir() + "/storage/session/"

type Session struct {
	SessionId string
	Data      map[string]interface{}
}

func DestroySession(clientMeta HttpClientMeta) {
	cookie := FindCookie(
		os.Getenv("SESSION_ID"),
		clientMeta.Cookies,
	)
	if cookie == nil {
		return
	}

	err := os.Remove(sessionDir + cookie.Value)
	if err != nil {
		fmt.Printf(
			"Failed to destroy session: %v\n",
			cookie,
		)
	}
}

func CreateSession() *Session {
	// Create hash
	hash := util.Generate(128)

	AddCookie(Cookie{
		Name:   os.Getenv("SESSION_ID"),
		Value:  hash,
		Path:   "/",
		Domain: os.Getenv("COOKIE_DOMAIN"),

		// 1 year
		MaxAge: 60 * 60 * 24 * 30 * 12,
	})

	// Generate session file
	handle, err := os.Create(sessionDir + hash)
	if err != nil {
		fmt.Printf("err = %s\n", err)
		return nil
	}

	_ = handle.Close()
	return &Session{
		SessionId: hash,
	}
}

func (session *Session) Write(key string, value string) {
	handle, err := os.OpenFile(sessionDir+session.SessionId, os.O_RDWR, 0644)
	if err != nil {
		// Session file has broken
		return
	}

	read, _ := ioutil.ReadAll(handle)
	data := map[string]interface{}{}

	// decode datum
	err = json.Unmarshal(read, &data)
	data[key] = value

	// Update session data
	session.Data = data

	// encrypt
	result, _ := json.Marshal(data)
	handle.Write(result)
	handle.Close()
}

func LoadSessionFromCookie(cookies []Cookie) *Session {
	cookie := FindCookie(os.Getenv("SESSION_ID"), cookies)
	if cookie == nil {
		return nil
	}

	handle, err := os.OpenFile(sessionDir+cookie.Value, os.O_RDWR, 0644)
	if err != nil {
		return nil
	}

	read, _ := ioutil.ReadAll(handle)
	data := map[string]interface{}{}

	// decode datum
	err = json.Unmarshal(read, &data)
	handle.Close()

	return &Session{
		SessionId: data["id"].(string),
		Data:      data,
	}
}
