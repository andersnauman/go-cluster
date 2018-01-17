package cluster

// NewMulticaster -
import (
	"net"
	"strconv"
	"time"
)

// Multicaster -
type Multicaster interface {
	Listen(cl *ClientList, errCh chan error)
	Annonce(cl *ClientList, errCh chan error)
}

// Multicast -
type Multicast struct {
	ServerInfo      ServerInfo
	MaxDatagramSize int
}

// NewMulticaster -
func NewMulticaster(address string, port int, protocol string) (m Multicast, err error) {
	m.ServerInfo = ServerInfo{
		Address:  address,
		Port:     port,
		Protocol: protocol,
	}
	m.MaxDatagramSize = 8192
	return
}

// Listen -
func (m Multicast) Listen(cl *ClientList, errCh chan error) {
	addr, err := net.ResolveUDPAddr("udp", m.ServerInfo.Address+":"+strconv.Itoa(m.ServerInfo.Port))
	if err != nil {
		errCh <- err
		return
	}
	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		errCh <- err
		return
	}
	defer l.Close()
	l.SetReadBuffer(m.MaxDatagramSize)
	for {
		clTemp := ClientList{}

		b := make([]byte, m.MaxDatagramSize)
		n, _, err := l.ReadFromUDP(b)
		if err != nil {
			errCh <- err
			continue
		}
		// DECRYPT
		if err = clTemp.unmarshalGob(b[:n]); err != nil {
			errCh <- err
			continue
		}
		if clTemp.Checksum == clTemp.computeChecksum() {
			cl.recvClients <- clTemp.Clients
		}
	}
}

// Annonce -
func (m Multicast) Annonce(cl *ClientList, errCh chan error) {
	w, err := dial(m.ServerInfo)
	if err != nil {
		return
	}
	for {
		time.Sleep(5 * time.Second)
		b, err := cl.marshalGob()
		if err != nil {
			errCh <- err
			continue
		}
		// ENCRYPT
		if err := write(w, b); err != nil {
			errCh <- err
		}
	}
}
