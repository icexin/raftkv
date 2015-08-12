package service

import (
	"time"

	"github.com/hashicorp/raft"
	"github.com/icexin/raftkv/proto"
	"github.com/juju/errors"
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
	if s.serv.raft.State() != raft.Leader {
		return errors.Errorf("not leader:%s", s.serv.raft.Leader())
	}
	v, err := s.serv.fsm.Get(req.Key, nil)
	if err != nil {
		return err
	}
	rep.Data = v
	return nil
}

// TODO forward
func (s *KVS) Write(req *proto.Request, rep *proto.Reply) error {
	if s.serv.raft.State() != raft.Leader {
		return errors.Errorf("not leader:%s", s.serv.raft.Leader())
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
