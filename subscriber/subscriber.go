package subscriber

import (
	"encoding/binary"
	"errors"
	"fmt"
	"lupinus/util"
	"lupinus/validator"
	"net"
	"os"
	"strconv"
)

const (
	chunkSize          = 8192
	protectedImageSize = 1024 * 1000 * 10
)

func SubscribeImageStream(connection net.Conn) ([]byte, [][]byte, int, error) {
	authKey := os.Getenv("AUTH_KEY")
	authKeySize := len(authKey)

	readAuthKey, err := util.ExpectToRead(connection, authKeySize)
	if readAuthKey == nil {
		return nil, nil, -1, err
	}

	// Compare the received auth key and settled auth key.
	if string(readAuthKey) != authKey {
		return nil, nil, -1, errors.New("Invalid auth key.")
	}

	// Receive frame size
	frameSize, errReceivingFrameSize := util.ExpectToRead(connection, 4)
	if frameSize == nil {
		return nil, nil, -1, errReceivingFrameSize
	}

	realFrameSize := int(binary.LittleEndian.Uint32(frameSize))

	if realFrameSize < 0 || protectedImageSize < realFrameSize {
		return nil, nil, -1, errors.New(
			"protected memory allocation. tried to alloc = " + strconv.Itoa(realFrameSize),
		)
	}

	realFrame, errReceivingRealFrame := util.ExpectToRead(connection, realFrameSize)

	if realFrame == nil {
		return nil, nil, -1, errReceivingRealFrame
	}

	if !validator.IsImageJpeg(realFrame) {
		fmt.Printf("image = %d\n", realFrameSize)
		return nil, nil, -1, errors.New("Does not match JPEG")
	}

	// Chunk the too long data.
	data, loops := util.Chunk(
		util.Byte2base64URI(
			realFrame,
		),
		chunkSize,
	)
	return realFrame, data, loops, nil
}
