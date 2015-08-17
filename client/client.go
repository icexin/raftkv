package raftkv

import (
	"math/rand"
	"net/rpc"
	"time"

	"gopkg.in/inconshreveable/log15.v2"

	"github.com/icexin/raftkv/proto"
)

type Client struct {
	addrs []string
	cli   *rpc.Client
	log   log15.Logger
}

func NewClient(addrs []string, log log15.Logger) *Client {
	if log == nil {
		log = log15.Root()
	}
	cli := &Client{
		addrs: addrs,
		log:   log,
	}
	cli.lookup()
	go cli.ping()
	return cli
}

func (c *Client) lookup() error {
	for {
		addr := c.addrs[rand.Intn(len(c.addrs))]
		cli, err := proto.DialMsgpack(addr, time.Second)
		if err == nil {
			c.cli = cli
			return nil
		}
		c.log.Error("connect", "error", err)
		time.Sleep(time.Second)
	}
	return nil
}

func (c *Client) ping() {
	for {
		err := c.cli.Call("KV.Ping", new(proto.Request), new(proto.Reply))
		if err != nil {
			c.cli.Close()
			c.lookup()
		}
		time.Sleep(time.Second)
	}
}

func (c *Client) Read(key []byte) ([]byte, error) {
	req := &proto.Request{
		Action: proto.OpRead,
		Key:    key,
	}
	rep := new(proto.Reply)
	err := c.cli.Call("KV.Read", req, rep)
	if err != nil {
		return nil, err
	}
	return rep.Data, nil
}

func (c *Client) Write(key, value []byte) error {
	req := &proto.Request{
		Action: proto.OpWrite,
		Key:    key,
		Data:   value,
	}
	rep := new(proto.Reply)
	return c.cli.Call("KV.Apply", req, rep)
}

func (c *Client) Delete(key []byte) error {
	req := &proto.Request{
		Action: proto.OpDelete,
		Key:    key,
	}
	rep := new(proto.Reply)
	return c.cli.Call("KV.Apply", req, rep)
}

func (c *Client) Close() error {
	return nil
}
