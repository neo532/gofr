package transport

import (
	"net"
	"strings"
)

// ClientIP resolves the real client IP from the direct peer address and a
// header lookup. It reads X-Forwarded-For (leftmost entry) then X-Real-IP,
// falling back to the peer address (the request's direct connection source).
//
// trustedProxies == nil means headers from any peer are trusted (the simple
// mode). A non-nil list restricts header trust to those proxy CIDRs: headers
// are honored only when the direct peer is in the list, so a client that
// connects directly cannot spoof its IP with forged headers.
func ClientIP(peerAddr string, getHeader func(string) string, trustedProxies []*net.IPNet) string {
	if trustedProxies == nil || isTrustedProxy(peerAddr, trustedProxies) {
		if ip := proxyHeaderIP(getHeader); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(peerAddr)
	if err != nil {
		return peerAddr
	}
	return host
}

// ParseTrustedProxies parses CIDR or single-IP strings into an IPNet list.
// Invalid entries are skipped. Used by each transport's TrustedProxies option.
func ParseTrustedProxies(cidrs ...string) []*net.IPNet {
	var out []*net.IPNet
	for _, c := range cidrs {
		if _, ipNet, err := net.ParseCIDR(c); err == nil {
			out = append(out, ipNet)
			continue
		}
		if ip := net.ParseIP(c); ip != nil {
			if ipv4 := ip.To4(); ipv4 != nil {
				out = append(out, &net.IPNet{IP: ipv4, Mask: net.CIDRMask(32, 32)})
			} else {
				out = append(out, &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)})
			}
		}
	}
	return out
}

func isTrustedProxy(peerAddr string, trustedProxies []*net.IPNet) bool {
	host, _, err := net.SplitHostPort(peerAddr)
	if err != nil {
		host = peerAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func proxyHeaderIP(getHeader func(string) string) string {
	// X-Forwarded-For can be "client, proxy1, proxy2"; the leftmost is the client.
	if xff := getHeader("X-Forwarded-For"); xff != "" {
		for _, part := range strings.Split(xff, ",") {
			if ip := net.ParseIP(strings.TrimSpace(part)); ip != nil {
				return ip.String()
			}
		}
	}
	if xri := getHeader("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
			return ip.String()
		}
	}
	return ""
}
