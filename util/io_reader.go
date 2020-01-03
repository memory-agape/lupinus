package util

import (
	"net"
)

func ExpectToRead(stream *net.Conn, expect int) ([]byte, error) {
	data := []byte{}
	remaining := expect
	for remaining > 0 {
		tmpRead := make([]byte, remaining)
		size, err := (*stream).Read(tmpRead)
		data = append(data, tmpRead...)
		if err != nil {
			return nil, err
		}

		remaining -= size
	}
	return data, nil
}