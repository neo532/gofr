/*
 * @abstract ip Conversion
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package lib

import (
	"encoding/binary"
	"net"
)

func Ip2long(ipStr string) uint32 {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	return binary.BigEndian.Uint32(ip)
}

func Long2ip(ipLong uint32) string {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipLong)
	ip := net.IP(ipByte)
	return ip.String()
}
