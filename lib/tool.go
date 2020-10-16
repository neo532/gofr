package lib

/*
 * @abstract tool
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"net"
	"strconv"
	"strings"
)

// CompareVersion returns the num after comparing two versions.
//ver1 > ver2 = 1
//ver1 < ver2 = -1
//ver1 = ver2 = 0
func CompareVersion(ver1, ver2 string) int {
	larger := 1
	smaller := -1
	v1 := strings.Split(ver1, ".")
	v2 := strings.Split(ver2, ".")
	v1Len := len(v1)
	v2Len := len(v2)

	//make sure that v1's length is larger
	if v1Len < v2Len {
		v1, v2 = v2, v1
		v1Len, v2Len = v2Len, v1Len
		larger, smaller = smaller, larger
	}

	v2MaxIndex := v2Len - 1
	var v1i, v2i int
	for i, v := range v1 {
		if i > v2MaxIndex {
			return larger
		}

		v1i, _ = strconv.Atoi(v)
		v2i, _ = strconv.Atoi(v2[i])
		if v1i > v2i {
			return larger
		}
		if v1i < v2i {
			return smaller
		}
	}
	return 0
}

// LocalIP returns the ip of local.
func LocalIP() string {
	def := "0.0.0.0"

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
