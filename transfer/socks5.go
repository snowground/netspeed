package transfer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const (
	socks5Version       = 0x05
	socks5AuthNone      = 0x00
	socks5AuthPassword  = 0x02
	socks5CmdUDPAssoc   = 0x03
	socks5AddrTypeIPv4  = 0x01
	socks5UDPHeaderIPv4 = 10
)

func parseSocks5Auth(addr string) (proxyAddr string, auth *proxy.Auth) {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		creds := addr[:i]
		proxyAddr = addr[i+1:]
		if j := strings.Index(creds, ":"); j >= 0 {
			auth = &proxy.Auth{
				User:     creds[:j],
				Password: creds[j+1:],
			}
		}
		return proxyAddr, auth
	}
	return addr, nil
}

func socks5Credentials(socks5Addr string) (proxyAddr, user, password string) {
	proxyAddr, auth := parseSocks5Auth(socks5Addr)
	if auth != nil {
		return proxyAddr, auth.User, auth.Password
	}
	return proxyAddr, "", ""
}

func dialTCPViaSocks5(socks5Addr, serverAddr, localAddr string) (*net.TCPConn, error) {
	proxyAddr, auth := parseSocks5Auth(socks5Addr)
	baseDialer := &net.Dialer{}
	if len(localAddr) > 0 {
		localTCPAddr, err := net.ResolveTCPAddr("tcp", localAddr)
		if err != nil {
			return nil, err
		}
		baseDialer.LocalAddr = localTCPAddr
	}
	dialer, err := proxy.SOCKS5("tcp", proxyAddr, auth, baseDialer)
	if err != nil {
		return nil, err
	}
	conn, err := dialer.Dial("tcp", serverAddr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

func socks5Handshake(tcp *net.TCPConn, user, password string) error {
	if user != "" && password != "" {
		if _, err := tcp.Write([]byte{socks5Version, 0x02, socks5AuthNone, socks5AuthPassword}); err != nil {
			return err
		}
	} else {
		if _, err := tcp.Write([]byte{socks5Version, 0x01, socks5AuthNone}); err != nil {
			return err
		}
	}

	buf := make([]byte, 32)
	n, err := tcp.Read(buf)
	if err != nil || n < 2 || buf[0] != socks5Version {
		return errors.New("invalid socks5 greeting reply")
	}

	switch buf[1] {
	case socks5AuthPassword:
		if user == "" || password == "" {
			return errors.New("socks5 proxy requires authentication")
		}
		authReq := make([]byte, 0, 3+len(user)+len(password))
		authReq = append(authReq, 0x01, byte(len(user)))
		authReq = append(authReq, user...)
		authReq = append(authReq, byte(len(password)))
		authReq = append(authReq, password...)
		if _, err := tcp.Write(authReq); err != nil {
			return err
		}
		n, err = tcp.Read(buf)
		if err != nil || n < 2 || buf[1] != socks5AuthNone {
			return errors.New("socks5 authentication failed")
		}
	case socks5AuthNone:
	default:
		return errors.New("no acceptable socks5 auth method")
	}
	return nil
}

func socks5UDPAssociate(tcp *net.TCPConn, proxyHost string) (*net.UDPAddr, error) {
	req := []byte{socks5Version, socks5CmdUDPAssoc, 0x00, socks5AddrTypeIPv4, 0, 0, 0, 0, 0, 0}
	if _, err := tcp.Write(req); err != nil {
		return nil, err
	}

	buf := make([]byte, 32)
	n, err := tcp.Read(buf)
	if err != nil || n < 10 || buf[0] != socks5Version || buf[1] != 0x00 {
		return nil, errors.New("socks5 UDP ASSOCIATE failed")
	}

	relayIP := net.IP(buf[4:8])
	relayPort := int(binary.BigEndian.Uint16(buf[8:10]))
	if relayIP.Equal(net.IPv4zero) {
		if host, _, err := net.SplitHostPort(proxyHost); err == nil {
			if ip := net.ParseIP(host); ip != nil {
				relayIP = ip.To4()
			}
		}
		if relayIP == nil || relayIP.Equal(net.IPv4zero) {
			if ip := net.ParseIP(proxyHost); ip != nil {
				relayIP = ip.To4()
			}
		}
	}
	if relayIP == nil || relayIP.Equal(net.IPv4zero) {
		return nil, errors.New("invalid socks5 UDP relay address")
	}
	return &net.UDPAddr{IP: relayIP, Port: relayPort}, nil
}

func buildSocks5UDPRequest(destIP net.IP, destPort int, payload []byte) []byte {
	ip4 := destIP.To4()
	if ip4 == nil {
		return nil
	}
	packet := make([]byte, socks5UDPHeaderIPv4+len(payload))
	packet[0] = 0x00
	packet[1] = 0x00
	packet[2] = 0x00
	packet[3] = socks5AddrTypeIPv4
	copy(packet[4:8], ip4)
	binary.BigEndian.PutUint16(packet[8:10], uint16(destPort))
	copy(packet[10:], payload)
	return packet
}

func stripSocks5UDPResponse(data []byte) ([]byte, error) {
	if len(data) < socks5UDPHeaderIPv4 {
		return nil, errors.New("socks5 UDP reply too short")
	}
	return data[socks5UDPHeaderIPv4:], nil
}

func UDPProbeViaSocks5(socks5Addr, targetHost string, targetPort int, payload []byte, timeout time.Duration) (time.Duration, error) {
	proxyAddr, user, password := socks5Credentials(socks5Addr)
	tcp, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return 0, err
	}
	tcpConn := tcp.(*net.TCPConn)
	defer tcpConn.Close()

	if err := socks5Handshake(tcpConn, user, password); err != nil {
		return 0, err
	}

	relayAddr, err := socks5UDPAssociate(tcpConn, proxyAddr)
	if err != nil {
		return 0, err
	}

	destIP := net.ParseIP(targetHost)
	if destIP == nil {
		ips, err := net.LookupIP(targetHost)
		if err != nil || len(ips) == 0 {
			return 0, fmt.Errorf("resolve target host %s: %w", targetHost, err)
		}
		destIP = ips[0]
	}

	udp, err := net.DialUDP("udp", nil, relayAddr)
	if err != nil {
		return 0, err
	}
	defer udp.Close()

	packet := buildSocks5UDPRequest(destIP, targetPort, payload)
	if packet == nil {
		return 0, errors.New("socks5 UDP only supports IPv4 targets")
	}

	start := time.Now()
	if err := udp.SetDeadline(start.Add(timeout)); err != nil {
		return 0, err
	}
	if _, err := udp.Write(packet); err != nil {
		return 0, err
	}

	buf := make([]byte, 1024)
	n, err := udp.Read(buf)
	if err != nil {
		return 0, err
	}
	if _, err := stripSocks5UDPResponse(buf[:n]); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
