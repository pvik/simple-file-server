package main

import (
	"net"
)

// Get preferred outbound ip of this machine
func getServerIP() (net.IP, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return net.IP{}, err
	} else {
		defer conn.Close()

		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP, nil
	}
}
