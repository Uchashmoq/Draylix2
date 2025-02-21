package client

import (
	"Draylix2/dlog"
	"Draylix2/network"
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"
)

const (
	HttpsProxy = iota
	Socks5Proxy
	HttpProxy
)

const (
	Ipv4 = iota
	Domain
)

type ProxyInfo struct {
	proxyType   byte
	addrType    byte
	addr        string
	initialData []byte
}

type ProxyClient struct {
	localAddr     string
	listener      net.Listener
	draylixConfig *network.DraylixConfig
}

func NewProxyClient(localAddr string, draylixConfig *network.DraylixConfig) *ProxyClient {
	return &ProxyClient{
		localAddr:     localAddr,
		listener:      nil,
		draylixConfig: draylixConfig,
	}
}

func (c *ProxyClient) Listen() error {
	listener, err := net.Listen("tcp", c.localAddr)
	if err != nil {
		return err
	}
	dlog.Info("proxy client is listening at %s", c.localAddr)
	c.listener = listener
	go c.accept()
	return nil
}

func (c *ProxyClient) accept() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			dlog.Error("failed to accept local connection: %v", err)
		}
		dlog.Debug("local %s connected", conn.RemoteAddr().String())
		c.handleLocalConn(conn)
	}

}

func (c *ProxyClient) handleLocalConn(conn net.Conn) {
	proxyInfo, err := getProxyInfo(conn)
	if err != nil {
		dlog.Error("failed to handle local connection: %v", err)
	}

}

func getProxyInfo(conn net.Conn) (*ProxyInfo, error) {
	buf := make([]byte, 4*1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	proxyType, err := parseProxyType(buf[:n])
	if err != nil {
		return nil, err
	}
	if proxyType == HttpProxy || proxyType == HttpsProxy {
		return parseHttpProxyInfo(buf[:n])
	}
	return parseSocks5ProxyInfo(conn)
}

func parseSocks5ProxyInfo(conn net.Conn) (*ProxyInfo, error) {
	_, err := conn.Write([]byte{5, 0})
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	addr, err := parseSocks5Addr(buf[:n])
	if err != nil {
		return nil, err
	}
	info := &ProxyInfo{
		proxyType: Socks5Proxy,
		addrType:  Domain,
		addr:      addr,
	}
	if network.IsValidIP(addr) {
		info.addrType = Ipv4
	}
	return info, nil
}

func parseProxyType(p []byte) (byte, error) {
	if p[0] == 5 {
		return Socks5Proxy, nil
	} else if strings.HasPrefix(string(p), "CONNECT") {
		return HttpsProxy, nil
	} else if isHttpProxy(p) {
		return HttpProxy, nil
	} else {
		return 0, fmt.Errorf("unknown proxy type")
	}
}

func isHttpProxy(p []byte) bool {
	var req string
	if len(p) > 10 {
		req = string(p[:10])
	} else {
		req = string(p)
	}
	return strings.HasPrefix(req, "GET") || strings.HasPrefix(req, "POST") || strings.HasPrefix(req, "PUT") || strings.HasPrefix(req, "HEAD") || strings.HasPrefix(req, "DELETE") || strings.HasPrefix(req, "OPTIONS") || strings.HasPrefix(req, "TRACE")
}

func parseSocks5Addr(data []byte) (string, error) {
	if len(data) < 5 {
		return "", fmt.Errorf("socks5 request is too short")
	}
	if data[0] != 0x05 {
		return "", fmt.Errorf("socks version error")
	}
	command := data[1]
	if command != 0x01 {
		return "", fmt.Errorf("unsurpported socks5 command : %d", command)
	}
	addressType := data[3]
	var targetHost string
	switch addressType {
	case 0x01: // IPv4 address
		if len(data) < 10 {
			return "", fmt.Errorf("socks5 address is too short")
		}
		targetHost = fmt.Sprintf("%d.%d.%d.%d", data[4], data[5], data[6], data[7])
	case 0x03: // Domain name
		domainLength := int(data[4])
		if len(data) < 5+domainLength+2 {
			return "", fmt.Errorf("socks5 domain is too short")
		}
		targetHost = string(data[5 : 5+domainLength])
	case 0x04: // IPv6 address (unsupported in this example)
		return "", fmt.Errorf("ipv6 is unsupported")
	default:
		return "", fmt.Errorf("unknown socks5 address type %d", addressType)
	}
	return targetHost, nil
}

func parseHttpProxyInfo(requestBytes []byte) (*ProxyInfo, error) {
	reader := bytes.NewReader(requestBytes)
	req, err := http.ReadRequest(bufio.NewReader(reader))
	if err != nil {
		return nil, err
	}
	proxyType := parseHttpProxyType(req)
	addr, addrType := parsHttpAddr(req)
	info := &ProxyInfo{
		proxyType: proxyType,
		addrType:  addrType,
		addr:      addr,
	}
	if proxyType == HttpProxy {
		info.initialData = requestBytes
	}
	return info, nil
}

func parsHttpAddr(req *http.Request) (string, byte) {
	if network.IsValidIP(req.Host) {
		return req.Host, Ipv4
	}
	return req.Host, Domain
}

func parseHttpProxyType(req *http.Request) byte {
	if req.Method == "CONNECT" {
		return HttpsProxy
	}
	return HttpProxy
}
