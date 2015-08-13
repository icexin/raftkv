package proto

import (
	"errors"
	"io"

	"github.com/soheilhy/cmux"
	"github.com/ugorji/go/codec"
)

const (
	RaftProto = 0
	RpcProto  = 1
)

var (
	ErrNoLeader = errors.New("no leader")
)

var (
	msgpackHandle = codec.MsgpackHandle{}
)

type Action int

const (
	Read Action = iota
	Write
)

type Request struct {
	Action Action
	Key    []byte
	Data   []byte
}

type Reply struct {
	Data []byte
}

func Marshal(msg interface{}) ([]byte, error) {
	var buf []byte
	enc := codec.NewEncoderBytes(&buf, &msgpackHandle)
	err := enc.Encode(msg)
	return buf, err
}

func Unmarshal(buf []byte, msg interface{}) error {
	dec := codec.NewDecoderBytes(buf, &msgpackHandle)
	return dec.Decode(msg)
}

type Item struct {
	Key  []byte
	Data []byte
}

func ByteMatcher(b byte) cmux.Matcher {
	return func(r io.Reader) bool {
		var buf [1]byte
		_, err := io.ReadFull(r, buf[:])
		if err != nil {
			return false
		}
		return buf[0] == b
	}
}
