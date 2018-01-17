package main

import (
	"fmt"
	"net"
	"os"

	cluster "github.com/andersnauman/go-cluster"
)

func main() {
	mc, _ := cluster.NewMulticaster("224.0.0.1", 9999, "udp")
	iface, err := net.InterfaceByName(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	cl, err := cluster.NewClientLister(iface)
	if err != nil {
		fmt.Println(err.Error())
	}
	errCh := make(chan error)
	go cl.VerifyClients()
	go mc.Annonce(&cl, errCh)
	//go mc.Listen(&cl, errCh)
	/*go func() {
		for {
			time.Sleep(5 * time.Second)
			if len(cl.Clients) > 0 {
				fmt.Printf("Clientlist:\n%+v\n", cl.Clients)
			}
		}
	}()*/
	for {
		err := <-errCh
		fmt.Println(err)
	}
}
