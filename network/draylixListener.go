package network

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
)

type DraylixConfig struct {
	GetPasswd           func(string) (string, error)
	HandleInvalidAccess func(net.Conn)
}

type DraylixListener struct {
	config   *DraylixConfig
	listener net.Listener
}

func ListenDraylixOverTls(address string, tlsConfig *tls.Config, draylixConfig *DraylixConfig) (*DraylixListener, error) {
	listen, err := tls.Listen("tcp", address, tlsConfig)
	if err != nil {
		return nil, err
	}
	draylixListener := &DraylixListener{
		config:   draylixConfig,
		listener: listen,
	}
	return draylixListener, nil
}

func (d *DraylixListener) Accept() (net.Conn, error) {
	conn, err := d.listener.Accept()
	if err != nil {
		return nil, err
	}
	userId, passwd, err := d.auth(conn)
	if err != nil {
		d.config.HandleInvalidAccess(conn)
		return nil, err
	}
	return &DraylixConn{
		UserId:    userId,
		Passwd:    passwd,
		transport: conn,
	}, nil
}

func (d *DraylixListener) Close() error {
	return d.listener.Close()
}

func (d *DraylixListener) Addr() net.Addr {
	return d.listener.Addr()
}

func (d *DraylixListener) auth(conn net.Conn) (string, string, error) {
	type1, userIdb, err := readMessage(conn)
	if err != nil {
		return "", "", err
	}
	if type1 != UserIdReq {
		return "", "", fmt.Errorf("invalid message type, expected: UserIdReq, got: %d", type1)
	}

	userId := string(userIdb)
	passwd, err := d.config.GetPasswd(userId)
	if err != nil {
		return "", "", err
	}
	challenge := newChallenge()
	err = writeMessage(conn, ChallengeRep, challenge)
	if err != nil {
		return "", "", err
	}

	type2, challengeReq, err := readMessage(conn)
	if err != nil {
		return "", "", err
	}
	if type2 != ChallengeReq {
		return "", "", fmt.Errorf("invalid message type, expected: ChallengeReq, got: %d", type2)
	}

	if !checkChallenge(challenge, passwd, challengeReq) {
		return "", "", fmt.Errorf("invalid challenge")
	}

	err = writeMessage(conn, AuthSuccess, []byte("success!"))
	return userId, passwd, err
}

func checkChallenge(challenge []byte, passwd string, req []byte) bool {
	return bytes.Equal(req, passwdChallenge(challenge, passwd))
}

func newChallenge() []byte {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes
}
