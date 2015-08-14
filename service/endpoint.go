package service

import (
	"time"

	"github.com/icexin/raftkv/proto"
)

type KVS struct {
	serv *Server
}

func NewKVS(serv *Server) *KVS {
	return &KVS{
		serv: serv,
	}
}

// TODO forward
func (s *KVS) Read(req *proto.Request, rep *proto.Reply) error {
	if done, err := s.serv.forward("KVS.Read", req, rep); done {
		return err
	}

	v, err := s.serv.fsm.Get(req.Key, nil)
	if err != nil {
		return err
	}
	rep.Data = v
	return nil
}

// TODO forward
func (s *KVS) Apply(req *proto.Request, rep *proto.Reply) error {
	if done, err := s.serv.forward("KVS.Read", req, rep); done {
		return err
	}

	buf, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	ret := s.serv.raft.Apply(buf, time.Second)
	err = ret.Error()
	if err != nil {
		return err
	}

	if ret.Response() != nil {
		return ret.Response().(error)
	}
	return nil
}

func (s *KVS) Ping(req *proto.Request, rep *proto.Reply) error {
	return nil
}
