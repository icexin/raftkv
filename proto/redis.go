package proto

import (
	"bufio"
	"io"
	"net"
	"net/rpc"
	"strings"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/juju/errors"
)

var methodMap = map[string]string{
	"ping": "KV.Ping",
	"get":  "KV.Read",
	"set":  "KV.Apply",
	"del":  "KV.Apply",
}

var actionMap = map[string]Action{
	"ping": OpPing,
	"get":  OpRead,
	"set":  OpWrite,
	"del":  OpDelete,
}

type RedisServerCodec struct {
	c io.Closer
	r *bufio.Reader
	w *bufio.Writer

	seq uint64
	msg *Message
	req Request
}

func NewRedisServerCodec(conn net.Conn) *RedisServerCodec {
	return &RedisServerCodec{
		c: conn,
		r: bufio.NewReader(conn),
		w: bufio.NewWriter(conn),
	}
}

func (c *RedisServerCodec) ReadRequestHeader(r *rpc.Request) error {
	err := c.innerReadRequestHeader(r)
	if err != nil && err != io.EOF {
		log15.Error("ReadRequestHeader", "error", err)
		WriteArbitrary(c.w, err)
		c.w.Flush()
	}
	return err
}

func (c *RedisServerCodec) innerReadRequestHeader(r *rpc.Request) (err error) {
	defer func() {
		if ret := recover(); ret != nil {
			err = errors.Errorf("Read Header:%s", ret)
		}
	}()

	m, err := bufioReadMessage(c.r)
	if err != nil {
		return
	}

	arr, err := m.Array()
	if err != nil {
		err = errors.Errorf("must by array type:%v", m.Type)
		return
	}

	op, _ := arr[0].Str()
	op = strings.ToLower(op)
	method, ok := methodMap[op]
	if !ok {
		err = errors.Errorf("method not supported:%s", op)
		return
	}
	r.ServiceMethod = method
	c.seq++
	r.Seq = c.seq
	c.msg = m
	return nil
}

func (c *RedisServerCodec) ReadRequestBody(x interface{}) error {
	err := c.innerReadRequestBody(x)
	if err != nil {
		log15.Error("ReadRequestBody", "error", err)
	}
	return err
}

func (c *RedisServerCodec) innerReadRequestBody(x interface{}) (err error) {
	defer func() {
		if ret := recover(); ret != nil {
			err = errors.Errorf("Read Body:%s", ret)
		}
	}()

	if x == nil {
		return nil
	}

	if c.msg == nil {
		return errors.New("empty message")
	}

	req, ok := x.(*Request)
	if !ok {
		return errors.New("body must be *Request")
	}

	arr, _ := c.msg.Array()
	op, _ := arr[0].Str()

	// clear message
	c.msg = nil

	req.Action = actionMap[op]

	// ping need zero param
	if req.Action != OpPing {
		key, _ := arr[1].Bytes()
		req.Key = key
	}

	switch op {
	case "set":
		value, _ := arr[2].Bytes()
		req.Data = value
	}
	c.req = *req

	return nil
}

func (c *RedisServerCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	err := c.innerWriteResponse(r, x)
	if err != nil {
		log15.Error("WriteResponse", "error", err)
	}
	return err
}

func (c *RedisServerCodec) innerWriteResponse(r *rpc.Response, x interface{}) error {
	if r.Error != "" {
		WriteArbitrary(c.w, errors.New(r.Error))
		return c.w.Flush()
	}

	rep, ok := x.(*Reply)
	if !ok {
		return errors.Errorf("response must be *Reply:%T", x)
	}

	switch c.req.Action {
	case OpPing:
		WriteArbitrary(c.w, "pong")
	case OpRead:
		WriteArbitrary(c.w, rep.Data)
	default:
		WriteArbitrary(c.w, "OK")
	}
	return c.w.Flush()
}

func (c *RedisServerCodec) Close() error {
	return c.c.Close()
}

func ServeRedis(l net.Listener, s *rpc.Server) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		codec := NewRedisServerCodec(conn)
		go s.ServeCodec(codec)
	}
}
