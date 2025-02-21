package network

import (
	"bytes"
	"fmt"
	"log"
	"testing"
)

func TestDialDraylixOverTls(t *testing.T) {
	buffer := bytes.Buffer{}
	err := writeMessage(&buffer, 'a', []byte("hello"))
	if err != nil {
		log.Fatalln(err)
	}
	messageType, data, err := readMessage(&buffer)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%v %v", messageType, string(data))

}
