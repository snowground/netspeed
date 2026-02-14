package client

import (
	"net"
	"strings"
	"time"
)

const DiscoveryMagic = "netspeed"

func DiscoverByBroadcast(discoveryPort string, timeout time.Duration) []string {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil
	}
	defer conn.Close()
	setUDPBroadcast(conn)
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil
	}
	dest, err := net.ResolveUDPAddr("udp4", "255.255.255.255:"+discoveryPort)
	if err != nil {
		return nil
	}
	if _, err := conn.WriteToUDP([]byte(DiscoveryMagic), dest); err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	var result []string
	buf := make([]byte, 128)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(buf[:n]))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		result = append(result, s)
	}
	return result
}
