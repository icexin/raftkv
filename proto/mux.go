package proto

import (
	"errors"
	"io"
	"net"

	"gopkg.in/inconshreveable/log15.v2"
)

var (
	ErrListenerClosed = errors.New("listener closed")
)

type listener struct {
	net.Listener
	ch chan net.Conn
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-l.ch
	if !ok {
		return nil, ErrListenerClosed
	}
	return c, nil
}

type Mux struct {
	m   map[byte]chan net.Conn
	l   net.Listener
	log log15.Logger
}

func NewMux(l net.Listener, log log15.Logger) *Mux {
	if log == nil {
		log = log15.Root()
	}
	return &Mux{
		m:   make(map[byte]chan net.Conn),
		l:   l,
		log: log,
	}
}

// Handle register a Listener to Mux
// When a connection reached, Mux will compare first byte to proto,
// if equal, the corresponding Listener
// Note: the first byte will consume from connection
func (m *Mux) Handle(proto byte) net.Listener {
	ch := make(chan net.Conn)
	m.m[proto] = ch
	return &listener{
		Listener: m.l,
		ch:       ch,
	}
}

func (m *Mux) Serve() error {
	defer func() {
		for _, ch := range m.m {
			close(ch)
		}
	}()

	for {
		conn, err := m.l.Accept()
		if err != nil {
			m.log.Error("accept", "error", err)
			continue
		}

		// read first byte
		var b [1]byte
		_, err = io.ReadFull(conn, b[:])
		if err != nil {
			conn.Close()
			continue
		}

		// find matcher
		proto := b[0]
		ch, ok := m.m[proto]
		if ok {
			go func() {
				ch <- conn
			}()
			continue
		}

		// none matched, close connection
		m.log.Debug("nothing matched", "proto", proto)
		conn.Close()
	}

	return nil
}
