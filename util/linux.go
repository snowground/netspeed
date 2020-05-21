//+build linux

package util

import (
	"errors"
	"net"
	// "log"
	// "net"
	// "strings"
	// "syscall"
)

// func getlocalip(conn *net.TCPConn) string {
// 	return strings.Split(conn.LocalAddr().String(), ":")[0]
// }
func BindToDevice(conn *net.TCPConn) error {
	// localip := getlocalip(*conn)
	// f, _ := conn.File()
	// fd := int(f.Fd())

	// intfs, _ := net.Interfaces()
	// for _, intf := range intfs {
	// 	addrs, _ := intf.Addrs()
	// 	for _, addr := range addrs {
	// 		if ipnet, ok := addr.(*net.IPNet); ok {
	// 			if ipnet.IP.String() == localip {
	// 				err := syscall.SetsockoptString(fd, syscall.SOL_SOCKET,
	// 					syscall.SO_BINDTODEVICE, intf.Name)
	// 				if err == nil {
	// 					log.Printf("binding eth:%s ip:%s ok", intf.Name, localip)
	// 				} else {
	// 					log.Printf("binding eth:%s ip:%s err:%s", intf.Name, localip, err.Error())
	// 				}
	// 				return err
	// 			}
	// 		}
	// 	}
	// }
	return errors.New("not found")
}
