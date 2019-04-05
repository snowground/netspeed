package main

import (
	"fmt"
	"log"
	"net"
	"netspeed/protocol"
)

func handle_read(c net.Conn, blocksize uint32) {
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
func handle_write(c net.Conn, blocksize uint32) {
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
func handleConn(c net.Conn) {
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
		handle_read(c, header.DataLen)
		break
	case protocol.HEADER_FUNC_WRITE:
		handle_write(c, header.DataLen)
		break
	default:
		log.Printf("header.Func:%08x addr:%s", header.Func, c.RemoteAddr())
		break
	}

}

func main() {
	listen, err := net.Listen("tcp", ":8888")
	if err != nil {
		fmt.Println("listen error: ", err)
		return
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
			break
		}

		// start a new goroutine to handle the new connection
		go handleConn(conn)
	}
}
