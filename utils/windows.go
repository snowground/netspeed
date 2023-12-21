//go:build windows
// +build windows

package utils

import "net"

func BindToDevice(conn net.Conn) error {
	return nil
}
