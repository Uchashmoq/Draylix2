package main

import (
	"Draylix2/network"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {
	//testKey()
	//testTUI()
	//network.TestAuth()
	//testConn()
}

func testConn() {
	tlsConfig1 := &tls.Config{
		InsecureSkipVerify: false,
	}

	certFile := "server-cert.pem"
	keyFile := "server-key.pem"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to load key pair: %v", err)
	}

	tlsConfig2 := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	dcfg := &network.DraylixConfig{
		GetPasswd:           tp,
		HandleInvalidAccess: hia,
	}
	addr := "127.0.0.1:16666"
	li, err := network.ListenDraylixOverTls(addr, tlsConfig2, dcfg)
	if err != nil {
		log.Fatalln(err)
	}
	go func() {
		for {
			conna, err2 := li.Accept()
			if err2 != nil {
				log.Fatalln(err2)
			}

			go sv(conna)
		}
	}()

	conn, err := network.DialDraylixOverTls("xjp", "12345678", addr, tlsConfig1)
	if err != nil {
		log.Fatalln(err)
	}
	i := 0
	for {
		i++
		conn.Write([]byte(fmt.Sprintf("msg: %d", i)))
		time.Sleep(1 * time.Second)
	}

}

func sv(conn net.Conn) {
	buf := make([]byte, 200)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
		}
		fmt.Printf("server: %s", string(buf[:n]))
		conn.Write(buf[:n])
	}
}

func tp(id string) (string, error) {
	return "12345678", nil
}

func hia(conn net.Conn) {
	log.Fatalln("invalid access")
}

func testTUI() {
	tui := ui.NewClientTUI()
	tui.SetNode("Singapore vultr1 47.108.118.112")
	tui.SetAddress("127.0.0.1:9988")
	go func() {
		i := 0
		for {
			i++
			tui.Log(fmt.Sprintf("this is log : %d", i))
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		for {
			time.Sleep(200 * time.Millisecond)
			tui.UpChan <- 1000
		}
	}()

	tui.Run()
}
