package proto

import (
	"net"
	"net/rpc"
	"time"

	"github.com/ugorji/go/codec"
)

func ServeMsgpack(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		codec := codec.MsgpackSpecRpc.ServerCodec(conn, &msgpackHandle)
		go rpc.ServeCodec(codec)
	}
}

func DialMsgpack(addr string, timeout time.Duration) (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write([]byte{RpcProto})
	if err != nil {
		conn.Close()
		return nil, err
	}
	codec := codec.MsgpackSpecRpc.ClientCodec(conn, &msgpackHandle)
	return rpc.NewClientWithCodec(codec), nil
}
