package network

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type DraylixConn struct {
	UserId    string
	Passwd    string
	transport net.Conn
}

func DialDraylixOverTls(userId, passwd, addr string, config *tls.Config) (*DraylixConn, error) {
	tlsConn, err := tls.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	err = clientAuth(tlsConn, userId, passwd)
	if err != nil {
		_ = tlsConn.Close()
		return nil, fmt.Errorf("draylix authentication failed: %s", err)
	}
	return &DraylixConn{
		UserId:    userId,
		Passwd:    passwd,
		transport: tlsConn,
	}, nil
}

func (d *DraylixConn) Read(b []byte) (n int, err error) {
	return d.transport.Read(b)
}

func (d *DraylixConn) Write(b []byte) (n int, err error) {
	return d.transport.Write(b)
}

func (d *DraylixConn) Close() error {
	return d.transport.Close()
}

func (d *DraylixConn) LocalAddr() net.Addr {
	return d.transport.LocalAddr()
}

func (d *DraylixConn) RemoteAddr() net.Addr {
	return d.transport.RemoteAddr()
}

func (d *DraylixConn) SetDeadline(t time.Time) error {
	return d.transport.SetDeadline(t)
}

func (d *DraylixConn) SetReadDeadline(t time.Time) error {
	return d.transport.SetReadDeadline(t)
}

func (d *DraylixConn) SetWriteDeadline(t time.Time) error {
	return d.transport.SetWriteDeadline(t)
}
