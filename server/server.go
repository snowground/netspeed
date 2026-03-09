package server

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"

	"netspeed/protocol"
	"netspeed/transfer"
)

const DiscoveryPort = "1235"

func handle_read(c transfer.Conn, blocksize uint32) {
	log.Printf("handle_read from conn:%s blocksize:%d", c.RemoteAddr(), blocksize)
	var buf = make([]byte, blocksize)
	for {
		n, err := c.Write(buf)
		if err != nil || n < 0 {
			log.Println("conn write error:", err)
			return
		}
	}
}
func handle_write(c transfer.Conn, blocksize uint32) {
	log.Printf("handle_write from conn:%s blocksize:%d", c.RemoteAddr(), blocksize)
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

	// read from the connection
	var buf = make([]byte, 1024)

	//	n, err := c.Read(buf)

	n, err := c.Read(buf)
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
	case protocol.HEADER_FUNC_READ:
		c.SetBuffer(int(header.DataLen), int(header.DataLen))
		handle_read(c, header.DataLen)
		break
	case protocol.HEADER_FUNC_WRITE:
		c.SetBuffer(int(header.DataLen), int(header.DataLen))
		handle_write(c, header.DataLen)
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
	log.Printf("udp echo listening on :%s (same port as TCP)", port)
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
		fmt.Println("transferType error: ", transferType)
		wg.Done()
		return
	}

	if err != nil {
		fmt.Println("listen error: ", err)
		wg.Done()
		return
	}
	servicePort := parsePortFromAddress(address)
	if servicePort != "" {
		go ServeDiscovery(DiscoveryPort, servicePort)
		go ServeUDPEcho(servicePort)
	}
	log.Printf("listen:%s %s", address, transferType)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
			break
		}
		go handleConn(conn)
	}
	wg.Done()
}
