package transfer
import (
	"net"
	"time"
)
type TcpConn struct {
	conn *net.TCPConn
}

type TcpListener struct{
	lis *net.TCPListener
}
func TcpServer(localAddr string) (*TcpListener, error){
	addr, err := net.ResolveTCPAddr("tcp", localAddr)
	if  err != nil{
		return nil,err
	}
	listen, err := net.ListenTCP("tcp", addr)
	if  err!= nil {
		return nil,err
	}
	return &TcpListener{lis : listen},nil
}
func (l*TcpListener) Accept() (Conn,error) {
	c,err:=l.lis.AcceptTCP()
	if err != nil{
		return nil,err
	}
	return &TcpConn{conn:c},nil
}
func TcpConnect(serverAddr string, localAddr string) (*TcpConn,error) {
	serveraddr, serr := net.ResolveTCPAddr("tcp", serverAddr)
	if serr != nil {
		return nil, serr
	}
	var localaddr *net.TCPAddr = nil
	var lerr error = nil
	if len(localAddr) > 0 {
		localaddr, lerr = net.ResolveTCPAddr("tcp", localAddr)
		if lerr != nil {
			return  nil, serr
		}
	}
	conn, err := net.DialTCP("tcp", localaddr, serveraddr)
	if err != nil {
		return  nil, err
	}
	return  &TcpConn{conn:conn}, nil
}

func (c *TcpConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// Write implements the Conn Write method.
func (c *TcpConn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *TcpConn) SetBuffer(readBytes int,writeBytes int) error {
	if err := c.conn.SetReadBuffer(readBytes); err != nil {
		return err
	}
	if err := c.conn.SetWriteBuffer(writeBytes); err != nil {
		return err
	}
	return nil
}

func (c *TcpConn) Close() error{
	return c.conn.Close()
}

func (c *TcpConn) SetDeadline(readt time.Time,writet time.Time) error{
	if err := c.conn.SetReadDeadline(readt); err != nil {
		return err
	}
	if err := c.conn.SetWriteDeadline(writet); err != nil {
		return err
	}
	return nil
}
func (c *TcpConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}