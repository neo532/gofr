package lib

/*
 * @abstract ip Conversion
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"encoding/binary"
	"net"
	"strings"
)

// IP2long converts the unit32 to the ip.
func IP2long(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

// Long2ip converts the ip a the uint32.
func Long2ip(ipLong uint32) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipLong)
	ip := net.IP(ipByte)
	return ip.String()
}

// LocalIP returns the ip of local.
func LocalIP() string {
	def := "127.0.0.1"

	eth0, err := net.InterfaceByName("eth0")
	if err != nil {
		return def
	}

	if ipList, err := eth0.Addrs(); err == nil {
		for _, v := range ipList {
			if ip := strings.Split(v.String(), "/"); len(ip) > 1 {
				return ip[0]
			}
		}
	}
	return def
}
