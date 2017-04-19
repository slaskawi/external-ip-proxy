package proxy

import (
	"io"
	"net"
	"fmt"
	log "github.com/slaskawi/external-ip-proxy/logging"
)

type dataDirection string

const (
	Inbound dataDirection = "inbound"
	Outbound dataDirection = "outbound"
)

var logger = log.NewLogger("proxy")

// Proxy - Manages a Proxy connection, piping data between local and remote.
type Proxy struct {
	SentBytes     uint64
	ReceivedBytes uint64

	LocalAddress  string
	RemoteAddress string

	error error
	errsig        chan bool

	// Settings
	NoDelay   bool
	OutputHex bool
}

func NewProxy(LocalAddress string, RemoteAddress string) *Proxy {
	return &Proxy{
		LocalAddress:  LocalAddress,
		RemoteAddress: RemoteAddress,
	}
}

// Start - open connection to remote and start proxying data.
func (p *Proxy) Start() error {
	laddr, err := net.ResolveTCPAddr("tcp", p.LocalAddress)
	if err != nil {
		return fmt.Errorf("Failed to resolve local address: %s", err)
	}
	raddr, err := net.ResolveTCPAddr("tcp", p.RemoteAddress)
	if err != nil {
		return fmt.Errorf("Failed to resolve remote address: %s", err)
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return fmt.Errorf("Failed to open local port to listen: %s", err)
	}
	defer listener.Close()
	logger.Info("Proxing from %q to %q", laddr, raddr)

	for {
		lconn, err := listener.AcceptTCP()
		defer lconn.Close()

		//connect to remote
		rconn, err := net.DialTCP("tcp", nil, raddr)
		rconn.SetNoDelay(p.NoDelay)
		if err != nil {
			return fmt.Errorf("Remote connection failed: %s", err)
		}
		defer rconn.Close()

		//display both ends
		logger.Info("Opened %s >>> %s", laddr.String(), raddr.String())

		//bidirectional copy
		go p.pipe(lconn, rconn, Inbound)
		go p.pipe(rconn, lconn, Outbound)

		//wait for close...
		<-p.errsig

		if p.error != nil {
			return p.error
		}

		fmt.Printf("Closed (%d bytes sent, %d bytes recieved)", p.SentBytes, p.ReceivedBytes)
	}

	return nil
}

func (p *Proxy) err(s string, err error) {
	if p.error != nil {
		//we already received an error... just move on.
		return
	}
	if err != io.EOF {
		p.error = fmt.Errorf(s, err)
	}
	p.errsig <- true
}

func (p *Proxy) pipe(src, dst io.ReadWriter, direction dataDirection) {
	var byteFormat string
	if p.OutputHex {
		byteFormat = "%x"
	} else {
		byteFormat = "%s"
	}

	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed '%s'\n", err)
			return
		}
		b := buff[:n]

		//show output
		logger.Info("%v >>> %v", direction, n)
		logger.Info(byteFormat, b)

		//write out result
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed '%s'\n", err)
			return
		}
		if direction == Outbound {
			p.SentBytes += uint64(n)
		} else {
			p.ReceivedBytes += uint64(n)
		}
	}
}
