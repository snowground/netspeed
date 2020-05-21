package transfer

import (
	"net"
	"time"
	"github.com/xtaci/kcp-go"
)
type KcpConn struct {
	conn *kcp.UDPSession
}

type KcpListener struct{
	lis *kcp.Listener
}
func KcpServer(localAddr string) (*KcpListener, error){
	lis, err := kcp.ListenWithOptions(localAddr, nil, 0, 0)
	return &KcpListener{lis : lis},err
}
func (l*KcpListener) Accept() (Conn,error) {
	c,err:=l.lis.AcceptKCP()
	if err != nil{
		return nil,err
	}
	return &KcpConn{conn:c},nil
}
func KcpConnect(serverAddr string, localAddr string) (*KcpConn,error) {
	kcpconn, err := kcp.DialWithOptions(serverAddr, nil, 0, 0)
	if err != nil {
		return  nil, err
	}
	return  &KcpConn{conn:kcpconn}, nil
}

func (c *KcpConn) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// Write implements the Conn Write method.
func (c *KcpConn) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *KcpConn) SetBuffer(readBytes int,writeBytes int) error {
	if err := c.conn.SetReadBuffer(readBytes); err != nil {
		return err
	}
	if err := c.conn.SetWriteBuffer(writeBytes); err != nil {
		return err
	}
	return nil
}

func (c *KcpConn) Close() error{
	return c.conn.Close()
}

func (c *KcpConn) SetDeadline(readt time.Time,writet time.Time) error{
	if err := c.conn.SetReadDeadline(readt); err != nil {
		return err
	}
	if err := c.conn.SetWriteDeadline(writet); err != nil {
		return err
	}
	return nil
}
func (c *KcpConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

