package network

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
)

func readn(reader io.Reader, n uint32) ([]byte, error) {
	buf := make([]byte, n)
	for k := uint32(0); k < n; {
		d, err := reader.Read(buf[k:])
		if err != nil {
			return nil, err
		}
		k += uint32(d)
	}
	return buf, nil
}

func writeMessage(writer io.Writer, msgType MessageType, data []byte) error {
	_, err := writer.Write(Head)
	if err != nil {
		return err
	}

	msgTypeBuf := []byte{byte(msgType)}
	_, err = writer.Write(msgTypeBuf)
	if err != nil {
		return err
	}

	err = writeUint32(writer, uint32(len(data)))
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}

func readMessage(reader io.Reader) (MessageType, []byte, error) {
	head := make([]byte, HeadLen)
	_, err := reader.Read(head)
	if err != nil {
		return 0, nil, err
	}
	if !isValidHead(head) {
		return 0, nil, fmt.Errorf("invalid head : %s", hex.EncodeToString(head))
	}

	msgType := make([]byte, 1)
	_, err = reader.Read(msgType)
	if err != nil {
		return 0, nil, err
	}

	dataLen, err := readUint32(reader)
	if err != nil {
		return 0, nil, err
	}
	data, err := readn(reader, dataLen)
	if err != nil {
		return 0, nil, err
	}

	return MessageType(msgType[0]), data, nil
}

func isValidHead(head []byte) bool {
	return bytes.Equal(head, Head)
}

func uint32ToBytes(val uint32) []byte {
	buffer := bytes.Buffer{}
	_ = writeUint32(&buffer, val)
	return buffer.Bytes()
}

func writeUint32(writer io.Writer, val uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, val)
	_, err := writer.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func readUint32(reader io.Reader) (uint32, error) {
	data := make([]byte, 4)
	n, err := reader.Read(data)
	if err != nil {
		return 0, err
	}
	if n != 4 {
		return 0, fmt.Errorf("failed to read uint32")
	}
	u := binary.BigEndian.Uint32(data)
	return u, nil
}
