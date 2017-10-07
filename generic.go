package cluster

import (
	"io"
	"net"
	"strconv"
)

// ServerInfo -
type ServerInfo struct {
	Address  string
	Port     int
	Protocol string
}

func dial(s ServerInfo) (w io.Writer, err error) {
	return net.Dial("udp", s.Address+":"+strconv.Itoa(s.Port))
}

func write(w io.Writer, b []byte) (err error) {
	_, err = w.Write(b)
	return
}
