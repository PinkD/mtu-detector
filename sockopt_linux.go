package main

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func setDF(conn *net.UDPConn) error {
	c, err := conn.SyscallConn()
	if err != nil {
		panic(err)
	}
	err = c.Control(func(fd uintptr) {
		err := unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, unix.IP_PMTUDISC_DO)
		if err != nil {
			panic(err)
		}
	})
	return nil
}

func isMsgSizeErr(err error) bool {
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			return err.Err == unix.EMSGSIZE
		}
	}
	return false
}
