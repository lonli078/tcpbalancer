package main

import (
	"container/ring"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type Backend struct {
	host string
	port string
}

func (b *Backend) String() string {
	return fmt.Sprintf("%s:%s", b.host, b.port)
}

type BackendManager struct {
	rlist *ring.Ring
	mu    sync.RWMutex
}

func (bm *BackendManager) Add(b *Backend) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	value := &ring.Ring{Value: b}
	if bm.rlist == nil {
		bm.rlist = value
	} else {
		bm.rlist = bm.rlist.Link(value).Next()
	}
}

func (bm *BackendManager) Choose() *Backend {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	if bm.rlist == nil {
		return nil
	}
	ret := bm.rlist.Value.(*Backend)
	bm.rlist = bm.rlist.Next()
	return ret
}

func copy(wc io.WriteCloser, r io.Reader) {
	defer wc.Close()
	io.Copy(wc, r)
}

func handleConnection(us net.Conn, backend string) {
	log.Printf(backend)
	ds, err := net.Dial("tcp", backend)
	if err != nil {
		log.Printf("failed to dial %s: %s", backend, err)
		us.Close()
		return
	}
	go copy(ds, us)
	go copy(us, ds)
}

func tcpBalance(bind string, bmanager *BackendManager) {
	log.Println("tcpBalance start")
	netlisten, err := net.Listen("tcp", bind)
	if err != nil {
		log.Printf("tcpBalance Listen error: %s", err)
		return

	}
	for {
		conn, err := netlisten.Accept()
		if err != nil {
			fmt.Errorf("tcpBalance accept error: %s", err)
			continue
		}
		go handleConnection(conn, bmanager.Choose().String())
	}
}

func main() {
	bmanager := &BackendManager{}
	bmanager.Add(&Backend{host: "192.168.0.100", port: "10002"})
	bmanager.Add(&Backend{host: "192.168.0.101", port: "10002"})
	tcpBalance(":8000", bmanager)
}
