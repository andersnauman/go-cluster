package cluster

import (
	"bytes"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"
)

// StatusCode - The type is a int
type StatusCode int

// Declaration of StatusCode:s, incremented by iota
const (
	Online StatusCode = iota
	Offline
	Pending
)

// ClientList ...
type ClientList struct {
	recvClients chan []Client
	m           sync.Mutex
	Clients     []Client
	Checksum    [20]byte
}

// Client -
type Client struct {
	IPv4   net.IP
	IPv6   net.IP
	Port   int
	Status StatusCode
}

// NewClientLister -
func NewClientLister(iface *net.Interface) (cl ClientList, err error) {
	cl.recvClients = make(chan []Client)
	var ifaces []net.Interface
	if iface == nil {
		ifaces, err = net.Interfaces()
		if err != nil {
			return
		}
	} else {
		ifaces = append(ifaces, *iface)
	}
	var addrs []net.Addr
	for i := range ifaces {
		addrs, err = ifaces[i].Addrs()
		if err != nil {
			return
		}
		var ip net.IP
		var IPv4, IPv6 net.IP
		for _, addr := range addrs {
			ip, _, err = net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}
			switch {
			case ip.To4() != nil:
				IPv4 = ip
			case ip.To16() != nil:
				IPv6 = ip
			}
		}
		if IPv4 == nil && IPv6 == nil {
			continue
		}
		cl.Clients = append(cl.Clients, Client{
			IPv4:   IPv4,
			IPv6:   IPv6,
			Port:   5984,
			Status: Pending,
		})
	}
	cl.Checksum = cl.computeChecksum()
	return
}

// VerifyClients -
func (cl *ClientList) VerifyClients() {
	for {
		select {
		case newClients := <-cl.recvClients:
			cl.compare(newClients)
		case <-time.After(5 * time.Second):
			for i, client := range cl.Clients {
				if client.Status != Pending {
					continue
				}
				// Make this more dynamic
				if _, err := net.Dial("tcp", fmt.Sprintf("%s:%d", client.IPv4.String(), client.Port)); err == nil {
					cl.m.Lock()
					cl.Clients[i].Status = Online
					cl.Checksum = cl.computeChecksum()
					cl.m.Unlock()
				}
			}
		}
	}
}

func (cl *ClientList) computeChecksum() [20]byte {
	return sha1.Sum([]byte(fmt.Sprintf("%#v", cl.Clients)))
}

func (cl *ClientList) compare(clients []Client) {
	for _, client := range clients {
		var IPv4, IPv6 net.IP
		if ip := net.ParseIP(client.IPv4.String()); ip != nil {
			IPv4 = ip
		}
		if ip := net.ParseIP(client.IPv6.String()); ip != nil {
			IPv6 = ip
		}
		// If neither IPv4 and IPv6 can be parsed, continue to the next record
		if IPv4 == nil && IPv6 == nil {
			continue
		}
		// If both IPv4 and IPv6 are found, continue to the next record.
		if cl.contains(IPv4) != false && cl.contains(IPv6) != false {
			continue
		}
		cl.m.Lock()
		addClient := true
		// If either IPv4 and IPv6 are not equal to the already
		// saved one, update information.
		for i, client := range cl.Clients {
			if client.IPv4.String() == IPv4.String() &&
				client.IPv6.String() != IPv6.String() {
				cl.Clients[i].IPv6 = IPv6
				addClient = false
			}
			if client.IPv4.String() != IPv4.String() &&
				client.IPv6.String() == IPv6.String() {
				cl.Clients[i].IPv4 = IPv4
				addClient = false
			}
		}
		// If the record is not known already, add it.
		if addClient == true {
			cl.Clients = append(cl.Clients, Client{
				IPv4:   IPv4,
				IPv6:   IPv6,
				Port:   client.Port,
				Status: Pending,
			})
		}
		cl.m.Unlock()
	}
}

func (cl *ClientList) contains(ip net.IP) bool {
	for _, client := range cl.Clients {
		if client.IPv4.String() == ip.String() ||
			client.IPv6.String() == ip.String() {
			return true
		}
	}
	return false
}

func (cl *ClientList) marshalGob() (b []byte, err error) {
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	cl.m.Lock()
	defer cl.m.Unlock()
	if err = enc.Encode(cl); err != nil {
		return nil, err
	}
	return network.Bytes(), err
}

func (cl *ClientList) unmarshalGob(b []byte) (err error) {
	var network bytes.Buffer
	enc := gob.NewDecoder(&network)
	if _, err = network.Write(b); err != nil {
		return
	}
	return enc.Decode(&cl)
}
