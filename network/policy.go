package network

import (
	"Draylix2/dlog"
	"encoding/json"
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"os"
	"strings"
)

const (
	LocationPolicy = "location"
	IPPolicy       = "ip"
	DomainPolicy   = "domain"

	UseProxy = 1
	Direct=0
)

type Policy struct {
	Type    string
	Value   string
	IsProxy int
}

type PolicySelector struct {
	policies []*Policy
	MMDB     *geoip2.Reader
}

func (ps *PolicySelector) LoadFromJson(file string) error {
	// 打开 JSON 文件
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	// 解析 JSON 数据
	decoder := json.NewDecoder(f)
	var policies []*Policy
	if err := decoder.Decode(&policies); err != nil {
		return err
	}

	// 赋值给 PolicySelector
	ps.policies = policies
	return nil
}

func (ps *PolicySelector) Select(remoteConn, localConn net.Conn, info *ProxyInfo) (net.Conn, error) {
	switch info.AddrType {
	case Ipv4:
		return ps.handshakeIpv4(remoteConn, localConn, info)

	}
}

func (ps *PolicySelector) handshakeIpv4(remoteConn, localConn net.Conn, info *ProxyInfo) (net.Conn, error) {
	policy := ps.findIpAndLocationPolicy(info.Addr)
	if policy.IsProxy==Direct {
		_=remoteConn.Close()
		proxyConn, err := ps.EstablishDirectConn(localConn, info)
		if err != nil {
			return nil, err
		}
		dlog.Info("%s",proxyLog(policy.IsProxy,localConn.RemoteAddr().String(), proxyConn.RemoteAddr().String())
	}


}

func proxyLog(proxy int, from string, to string) any {

}



func (ps *PolicySelector) findIpAndLocationPolicy(addr string) *Policy {
	for _, p := range ps.policies {
		if p.Type == IPPolicy {
			ok, err := matchIp(p.Value, addr)
			if err != nil {
				dlog.Error("policy error: %s", err)
			}
			if ok {
				return p
			}
		}

		if p.Type == LocationPolicy {
			ok, err := matchLocation(p.Value, addr, ps.MMDB)
			if err != nil {
				dlog.Error("policy error: %s", err)
			}
			if ok {
				return p
			}
		}

	}
	return nil
}

// 告诉本地连接开始发送正常数据
func (ps *PolicySelector) localReady(conn net.Conn, info *ProxyInfo) error {
	if info.ProxyType==HttpProxy{
		return nil
	}
	reply := info.getSuccessReply()
	_,err:=conn.Write(reply)
	return err
}

func (ps *PolicySelector) EstablishDirectConn(localConn net.Conn, info *ProxyInfo) (net.Conn, error) {
	dial, err := net.Dial("tcp", info.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to establish direct conn: %s",err)
	}
	err = ps.localReady(localConn, info)
	return dial,err
}

func matchLocation(locationName string, ip string, mmdb *geoip2.Reader) (bool, error) {
	// 去掉IP地址中的端口部分（如果有）
	ip = strings.Split(ip, ":")[0]

	// 查询IP地址的地理位置信息
	record, err := mmdb.City(net.ParseIP(ip))
	if err != nil {
		return false, fmt.Errorf("failed to query location: %v", err)
	}

	// 获取国家名称和城市名称
	countryName := record.Country.Names["en"] // 使用英文名称
	cityName := record.City.Names["en"]      // 使用英文名称

	// 判断是否匹配
	if strings.EqualFold(countryName, locationName) || strings.EqualFold(cityName, locationName) {
		return true, nil
	}

	// 如果不匹配，返回false
	return false, nil
}

func matchIp(cidr, ipv4 string) (bool, error) {
	// 去掉IP地址中的端口部分（如果有）
	ip := strings.Split(ipv4, ":")[0]

	// 解析CIDR网段
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}

	// 解析IPv4地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false, fmt.Errorf("can not parse Ip :%s", ip)
	}

	// 检查IP是否属于CIDR网段
	return ipNet.Contains(parsedIP), nil
}
