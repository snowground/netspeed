package server

import (
	"log"
	"net"
	"strings"
)

const DiscoveryMagic = "netspeed"

func localIPv4Strings() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.IsLoopback() {
				continue
			}
			ip := ipnet.IP.To4()
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

func ServeDiscovery(discoveryPort string, servicePort string) {
	addr, err := net.ResolveUDPAddr("udp4", ":"+discoveryPort)
	if err != nil {
		log.Println("discovery resolve:", err)
		return
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Println("discovery listen:", err)
		return
	}
	defer conn.Close()
	buf := make([]byte, 64)
	ips := localIPv4Strings()
	if len(ips) == 0 {
		log.Println("discovery: no local IPv4")
		return
	}
	log.Printf("discovery listening on UDP :%s (reply with service port %s)", discoveryPort, servicePort)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		if n < len(DiscoveryMagic) {
			continue
		}
		msg := strings.TrimSpace(string(buf[:n]))
		if msg != DiscoveryMagic {
			continue
		}
		for _, ip := range ips {
			reply := ip + ":" + servicePort
			_, _ = conn.WriteToUDP([]byte(reply), remote)
		}
	}
}
