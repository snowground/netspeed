package transfer

import (
	"net"
	"time"
)
type Conn interface {
	Read(b []byte) (int, error) 
	Write(b []byte) (int, error)
	SetBuffer(readBytes int,writeBytes int) error 
	SetDeadline(readt time.Time,writet time.Time) error
	RemoteAddr() net.Addr
	Close() error
}

type Listener interface {
	Accept() (Conn,error)
}