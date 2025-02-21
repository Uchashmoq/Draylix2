package network

import (
	"crypto/sha256"
	"fmt"
	"net"
)

const (
	UserIdReq = MessageType(iota)
	ChallengeRep
	ChallengeReq
	AuthSuccess
)

var (
	Head    = []byte("drlx2")
	HeadLen = len(Head)
)

type MessageType byte
type Challenge int

type AuthMessage struct {
	UserId    string
	Challenge Challenge
}

func clientAuth(conn net.Conn, userId, passwd string) error {
	err := writeMessage(conn, UserIdReq, []byte(userId))
	if err != nil {
		return err
	}
	messageType, challenge, err := readMessage(conn)
	if err != nil {
		return err
	}
	if messageType != ChallengeRep {
		return fmt.Errorf("expected message type %v, got %v", ChallengeRep, messageType)
	}

	sum := passwdChallenge(challenge, passwd)
	err = writeMessage(conn, ChallengeReq, sum)
	if err != nil {
		return err
	}

	messageType, _, err = readMessage(conn)
	if err != nil {
		return err
	}

	if messageType != AuthSuccess {
		return fmt.Errorf("authentication failed")
	}

	return nil
}

func passwdChallenge(challenge []byte, passwd string) []byte {
	//log.Printf("cha:%s , passwd:%s", hex.EncodeToString(challenge), passwd)
	hash := sha256.New()
	hash.Write(challenge)
	hash.Write([]byte(passwd))
	return hash.Sum(nil)
}

func BytesFormat(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float32(bytes)/1024.0)
	} else {
		return fmt.Sprintf("%.2f MB", float32(bytes)/1024.0/1024.0)
	}
}

func IsValidIP(host string) bool {
	// 使用 net.ParseIP 函数判断是否是有效的 IP 地址
	ip := net.ParseIP(host)
	return ip != nil
}
