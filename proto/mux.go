package proto

import (
	"bytes"
	"errors"
	"io"
	"net"

	"github.com/soheilhy/cmux"

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
	m      map[byte]chan net.Conn // protocol map
	cmuxch chan net.Conn          // cmux connection channel
	cmux   cmux.CMux              // cmux Handler
	l      net.Listener           // underlaying listener
	log    log15.Logger
}

func NewMux(l net.Listener, log log15.Logger) *Mux {
	if log == nil {
		log = log15.Root()
	}
	cmuxch := make(chan net.Conn)
	cmux := cmux.New(&listener{
		Listener: l,
		ch:       cmuxch,
	})

	go cmux.Serve()

	m := &Mux{
		m:      make(map[byte]chan net.Conn),
		cmuxch: cmuxch,
		cmux:   cmux,
		l:      l,
		log:    log,
	}

	return m
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

// Default will not comsume first byte
func (m *Mux) HandleThird(matcher ...cmux.Matcher) net.Listener {
	return m.cmux.Match(matcher...)
}

func (m *Mux) Close() error {
	return m.l.Close()
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
			return err
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

		// none matched, try default or close connection
		if m.cmuxch != nil {
			go func() {
				m.cmuxch <- newCombineConn(b[:], conn)
			}()
			continue
		}
		m.log.Debug("nothing matched", "proto", proto)
		conn.Close()
	}

	return nil
}

type combineConn struct {
	net.Conn
	r io.Reader
}

func newCombineConn(b []byte, conn net.Conn) *combineConn {
	r := io.MultiReader(bytes.NewReader(b), conn)
	return &combineConn{
		Conn: conn,
		r:    r,
	}
}

func (c *combineConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}
