package client

import (
	"Draylix2/dlog"
	"Draylix2/network"
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"net/http"
	"strings"
)

type ServerConfig struct {
	addr string
}

type ProxyClientConfig struct {
	LocalAddr    string
	ServerAddr   string
	UserId       string
	Passwd       string
	MMDBFile     string
	PoliciesFile string
	TlsConfig    *tls.Config
}

type ProxyClient struct {
	ClientConfig  *ProxyClientConfig
	listener      net.Listener
	proxySelector network.PolicySelector
}

func NewProxyClient(clientConfig *ProxyClientConfig) *ProxyClient {
	client := &ProxyClient{
		ClientConfig:  clientConfig,
		proxySelector: network.PolicySelector{},
	}
	db, err := geoip2.Open(clientConfig.MMDBFile)
	if err != nil {
		dlog.Warn("cannot open mmdb file: %s, %s", clientConfig.MMDBFile, err)
	}
	client.proxySelector.MMDB = db

	err = client.proxySelector.LoadFromJson(clientConfig.PoliciesFile)
	if err != nil {
		dlog.Warn("cannot open policies file %s, %s", clientConfig.PoliciesFile, err)
	}
	return client
}

func (c *ProxyClient) LoadPolicies(file string) error {
	return c.proxySelector.LoadFromJson(file)
}

func (c *ProxyClient) LoadMMDB(file string) error {
	db, err := geoip2.Open(file)
	if err != nil {
		return err
	}
	c.proxySelector.MMDB = db
	return nil
}

func (c *ProxyClient) Listen() error {
	listener, err := net.Listen("tcp", c.ClientConfig.LocalAddr)
	if err != nil {
		return err
	}
	dlog.Info("proxy client is listening at %s", c.ClientConfig.LocalAddr)
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
	drlxConn, err := network.DialDraylixOverTls(c.ClientConfig.UserId, c.ClientConfig.Passwd, c.ClientConfig.ServerAddr, c.ClientConfig.TlsConfig)
	if err != nil {
		dlog.Error("can not connect to server : %s", err)
	}
	proxyConn, err := c.proxySelector.Select(drlxConn, conn, proxyInfo)

}

func getProxyInfo(conn net.Conn) (*network.ProxyInfo, error) {
	buf := make([]byte, 4*1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	proxyType, err := parseProxyType(buf[:n])
	if err != nil {
		return nil, err
	}
	if proxyType == network.HttpProxy || proxyType == network.HttpsProxy {
		return parseHttpProxyInfo(buf[:n])
	}
	return parseSocks5ProxyInfo(conn)
}

func parseSocks5ProxyInfo(conn net.Conn) (*network.ProxyInfo, error) {
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
	info := &network.ProxyInfo{
		ProxyType: network.Socks5Proxy,
		AddrType:  network.Domain,
		Addr:      addr,
	}
	if network.IsValidIP(addr) {
		info.AddrType = network.Ipv4
	}
	return info, nil
}

func parseProxyType(p []byte) (byte, error) {
	if p[0] == 5 {
		return network.Socks5Proxy, nil
	} else if strings.HasPrefix(string(p), "CONNECT") {
		return network.HttpsProxy, nil
	} else if isHttpProxy(p) {
		return network.HttpProxy, nil
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

func parseHttpProxyInfo(requestBytes []byte) (*network.ProxyInfo, error) {
	reader := bytes.NewReader(requestBytes)
	req, err := http.ReadRequest(bufio.NewReader(reader))
	if err != nil {
		return nil, err
	}
	proxyType := parseHttpProxyType(req)
	addr, addrType := parsHttpAddr(req)
	info := &network.ProxyInfo{
		ProxyType: proxyType,
		AddrType:  addrType,
		Addr:      addr,
	}
	if proxyType == network.HttpProxy {
		info.InitialData = requestBytes
	}
	return info, nil
}

func parsHttpAddr(req *http.Request) (string, byte) {
	if network.IsValidIP(req.Host) {
		return req.Host, network.Ipv4
	}
	return req.Host, network.Domain
}

func parseHttpProxyType(req *http.Request) byte {
	if req.Method == "CONNECT" {
		return network.HttpsProxy
	}
	return network.HttpProxy
}
