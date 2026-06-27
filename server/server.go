package server

import (
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"netspeed/protocol"
	"netspeed/transfer"
)

const DiscoveryPort = "1235"

var (
	listener      transfer.Listener
	activeConns   []transfer.Conn
	activeConnsMu sync.Mutex
)

func registerConn(c transfer.Conn) {
	activeConnsMu.Lock()
	activeConns = append(activeConns, c)
	activeConnsMu.Unlock()
}

func unregisterConn(c transfer.Conn) {
	activeConnsMu.Lock()
	for i, ac := range activeConns {
		if ac == c {
			activeConns = append(activeConns[:i], activeConns[i+1:]...)
			break
		}
	}
	activeConnsMu.Unlock()
}

func Stop() {
	if listener != nil {
		listener.Close()
		listener = nil
	}
	activeConnsMu.Lock()
	for _, c := range activeConns {
		c.Close()
	}
	activeConns = nil
	activeConnsMu.Unlock()
}

func handle_download(c transfer.Conn, blocksize uint32) {
	log.Printf("handle_download from conn:%s blocksize:%d", c.RemoteAddr(), blocksize)
	var buf = make([]byte, blocksize)
	for {
		n, err := c.Write(buf)
		if err != nil || n < 0 {
			log.Println("conn write error:", err)
			return
		}
	}
}
func handle_upload(c transfer.Conn, blocksize uint32) {
	log.Printf("handle_upload from conn:%s blocksize:%d", c.RemoteAddr(), blocksize)
	var buf = make([]byte, blocksize)
	for {
		n, err := c.Read(buf)
		if err != nil || n < 0 {
			log.Println("conn read error:", err)
			return
		}
	}
}
func handleConn(c transfer.Conn) {
	defer c.Close()
	registerConn(c)
	defer unregisterConn(c)

	var buf = make([]byte, protocol.HeaderSize)

	n, err := io.ReadFull(c, buf)
	if err != nil {
		log.Println("conn read error: ", err, c.RemoteAddr())
		return
	}
	err, header := protocol.Data2header(buf, n)
	if err != nil {
		log.Printf("err:%s addr:%s", err, c.RemoteAddr())
		return
	}
	switch header.Func {
	case protocol.HEADER_FUNC_DOWNLOAD:
		c.SetBuffer(int(header.DataLen), int(header.DataLen))
		handle_download(c, header.DataLen)
		break
	case protocol.HEADER_FUNC_UPLOAD:
		c.SetBuffer(int(header.DataLen), int(header.DataLen))
		handle_upload(c, header.DataLen)
		break
	default:
		log.Printf("header.Func:%08x addr:%s", header.Func, c.RemoteAddr())
		break
	}

}

func parsePortFromAddress(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return ""
	}
	if _, err := strconv.Atoi(port); err != nil {
		return ""
	}
	return port
}

func ServeUDPEcho(port string) {
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum <= 0 {
		return
	}
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: portNum}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Printf("udp echo listen error: %v", err)
		return
	}
	defer conn.Close()
	log.Printf("udp echo listening on :%s (service port + 1)", port)
	buf := make([]byte, 256)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if n > 0 {
			_, _ = conn.WriteToUDP(buf[:n], remote)
		}
	}
}

func ServerMain(address string, transferType string, wg *sync.WaitGroup) {
	var l transfer.Listener
	var err error

	switch transferType {
	case "tcp":
		l, err = transfer.TcpServer(address)
	case "kcp":
		l, err = transfer.KcpServer(address)
	default:
		log.Printf("transferType error: %s", transferType)
		wg.Done()
		return
	}

	if err != nil {
		log.Printf("listen error: %v", err)
		wg.Done()
		return
	}
	listener = l
	servicePort := parsePortFromAddress(address)
	if servicePort != "" {
		go ServeDiscovery(DiscoveryPort, servicePort)
		if portNum, err := strconv.Atoi(servicePort); err == nil {
			go ServeUDPEcho(strconv.Itoa(portNum + 1))
		}
	}
	log.Printf("listen:%s %s", address, transferType)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			break
		}
		go handleConn(conn)
	}
	wg.Done()
}
