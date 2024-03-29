package server

import (
	"fmt"
	"log"
	"sync"

	"netspeed/protocol"
	"netspeed/transfer"
)

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
