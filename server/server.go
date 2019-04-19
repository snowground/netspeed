package server

import (
	"fmt"
	"log"
	"net"
	"netspeed/protocol"
	"netspeed/util"
	"sync"
)

func handle_read(c net.TCPConn, blocksize uint32) {
	log.Printf("handle_read from conn:%s blocksize:%d", c.RemoteAddr(), blocksize)
	var buf = make([]byte, blocksize)
	for {
		n, err := c.Write(buf)
		if err != nil || n < 0 {
			log.Println("conn read error:", err)
			return
		}
	}
}
func handle_write(c net.TCPConn, blocksize uint32) {
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
func handleConn(c net.TCPConn) {
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
		c.SetWriteBuffer(int(header.DataLen))
		handle_read(c, header.DataLen)
		break
	case protocol.HEADER_FUNC_WRITE:
		c.SetReadBuffer(int(header.DataLen))
		handle_write(c, header.DataLen)
		break
	default:
		log.Printf("header.Func:%08x addr:%s", header.Func, c.RemoteAddr())
		break
	}

}

func ServerMain(address string, wg *sync.WaitGroup) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		fmt.Println("listen error: ", err)
		wg.Done()
		return
	}
	log.Printf("listen:%s", address)
	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			fmt.Println("accept error: ", err)
			break
		}

		// start a new goroutine to handle the new connection
		util.BindToDevice(conn)
		go handleConn(*conn)
	}
	wg.Done()
}
