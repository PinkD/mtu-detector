package main

import (
	"net"
	"os"

	"golang.org/x/sys/windows"
)

// ref: https://github.com/database64128/swgp-go/blob/main/conn/conn_windows.go

const (
	IP_MTU_DISCOVER   = 71
	IPV6_MTU_DISCOVER = 71
)

// enum PMTUD_STATE from ws2ipdef.h
const (
	IP_PMTUDISC_NOT_SET = iota
	IP_PMTUDISC_DO
	IP_PMTUDISC_DONT
	IP_PMTUDISC_PROBE
	IP_PMTUDISC_MAX
)

func setDF(conn *net.UDPConn) error {
	c, err := conn.SyscallConn()
	if err != nil {
		panic(err)
	}
	err = c.Control(func(fd uintptr) {
		h := windows.Handle(fd)
		// TODO: set for ipv6
		err := windows.SetsockoptInt(h, windows.IPPROTO_IP, IP_MTU_DISCOVER, IP_PMTUDISC_DO)
		if err != nil {
			panic(err)
		}
	})
	return nil
}

func isMsgSizeErr(err error) bool {
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			return err.Err == windows.WSAEMSGSIZE
		}
	}
	return false
}
