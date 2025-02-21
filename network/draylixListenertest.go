package network

import (
	"bytes"
	"log"
	"net"
	"sync"

	"time"
)

func gp(id string) (string, error) {
	return "12345678", nil
}

func hia(conn net.Conn) {
	log.Printf("invalid access")
}

type testConn struct {
	mutex sync.Mutex
	peer  *testConn
	buf   bytes.Buffer
}

func newTestConns() (*testConn, *testConn) {
	c1 := &testConn{
		mutex: sync.Mutex{},
		peer:  nil,
		buf:   bytes.Buffer{},
	}
	c2 := &testConn{
		mutex: sync.Mutex{},
		peer:  nil,
		buf:   bytes.Buffer{},
	}
	c1.peer = c2
	c2.peer = c1
	return c1, c2
}

func (tc *testConn) Read(b []byte) (n int, err error) {

check:
	if tc.buf.Len() < len(b) {
		time.Sleep(100 * time.Millisecond)
		goto check
	}

	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	return tc.buf.Read(b)
}

func (tc *testConn) Write(b []byte) (n int, err error) {
	tc.peer.mutex.Lock()
	defer tc.peer.mutex.Unlock()
	return tc.peer.buf.Write(b)
}

func (tc *testConn) Close() error {
	//TODO implement me
	panic("implement me")
}

func (tc *testConn) LocalAddr() net.Addr {
	//TODO implement me
	panic("implement me")
}

func (tc *testConn) RemoteAddr() net.Addr {
	//TODO implement me
	panic("implement me")
}

func (tc *testConn) SetDeadline(t time.Time) error {
	//TODO implement me
	panic("implement me")
}

func (tc *testConn) SetReadDeadline(t time.Time) error {
	//TODO implement me
	panic("implement me")
}

func (tc *testConn) SetWriteDeadline(t time.Time) error {
	//TODO implement me
	panic("implement me")
}

func TestAuth() {
	li := DraylixListener{
		config: &DraylixConfig{
			GetPasswd:           gp,
			HandleInvalidAccess: hia,
		},
		listener: nil,
	}
	c1, c2 := newTestConns()
	go func() {
		id, passwd, err := li.auth(c1)
		if err != nil {
			log.Fatalln(err)
		} else {
			log.Printf("%s:%s auth", id, passwd)
		}
	}()

	err := clientAuth(c2, "xijinping", "123456789")
	if err != nil {
		log.Println(err)
	}
}
