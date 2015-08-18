package service

import (
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/icexin/raftkv/config"
	"github.com/icexin/raftkv/proto"
	"github.com/soheilhy/cmux"
)

type Server struct {
	cfg *config.Config

	// raft
	raftLayer *raftLayer
	raftTrans *raft.NetworkTransport
	raft      *raft.Raft

	rpcServer *rpc.Server

	fsm *FSM
	kvs *KVS
	mux *proto.Mux

	mutex sync.Mutex // protect conns
	conns map[string]*rpc.Client
}

func NewServer(cfg *config.Config) (*Server, error) {
	server := &Server{
		cfg:   cfg,
		conns: make(map[string]*rpc.Client),
	}

	// setup listener
	l, err := net.Listen("tcp", cfg.Server.Listen)
	if err != nil {
		return nil, err
	}

	// setup mux
	mux := proto.NewMux(l, nil)
	raftl := mux.Handle(proto.RaftProto)
	msgpackl := mux.Handle(proto.RpcProto)
	httpl := mux.HandleThird(cmux.HTTP1())
	redisl := mux.HandleThird(cmux.Any())

	// setup http server
	go http.Serve(httpl, nil)

	// setup rpc server
	kvs := NewKVS(server)
	rpcServer := rpc.NewServer()
	err = rpcServer.RegisterName("KV", kvs)
	if err != nil {
		return nil, err
	}
	// support msgpack rpc protocol
	go proto.ServeMsgpack(msgpackl, rpcServer)
	// support redis protocol
	go proto.ServeRedis(redisl, rpcServer)

	// setup raft transporter
	advertise, err := net.ResolveTCPAddr("tcp", cfg.Raft.Advertise)
	if err != nil {
		return nil, err
	}
	layer := NewRaftLayer(advertise, raftl)
	trans := raft.NewNetworkTransport(
		layer,
		5,
		time.Second,
		os.Stderr,
	)

	// setup raft fsm
	fsm, err := NewFSM(&cfg.DB)
	if err != nil {
		return nil, err
	}

	// setup raft
	raft, err := NewRaft(&cfg.Raft, fsm, trans)
	if err != nil {
		return nil, err
	}

	server.raftLayer = layer
	server.raftTrans = trans
	server.raft = raft
	server.rpcServer = rpcServer
	server.fsm = fsm
	server.kvs = kvs
	server.mux = mux
	return server, nil
}

func (s *Server) forward(method string, req, rep interface{}) (done bool, err error) {
	if s.raft.State() == raft.Leader {
		return false, nil
	}

	done = true

	leader := s.raft.Leader()
	if leader == "" {
		err = proto.ErrNoLeader
		return
	}

	s.mutex.Lock()
	cli, ok := s.conns[leader]
	s.mutex.Unlock()
	if !ok {
		// FIXME connection timeout hard code
		cli, err = proto.DialMsgpack(leader, time.Second*3)
		if err != nil {
			return
		}
		// cache connection
		s.mutex.Lock()
		s.conns[leader] = cli
		s.mutex.Unlock()
	}

	err = cli.Call(method, req, rep)
	if err != nil {
		// if is ServerError, do not close, otherwise close connection
		if _, ok := err.(rpc.ServerError); ok {
			return
		}
		cli.Close()
		s.mutex.Lock()
		delete(s.conns, leader)
		s.mutex.Unlock()
		return
	}

	return
}

func (s *Server) Serve() error {
	return s.mux.Serve()
}

func (s *Server) Close() error {
	s.raftLayer.Close()
	s.raftTrans.Close()
	ret := s.raft.Shutdown()
	// wait raft shutdown
	ret.Error()
	s.fsm.Close()
	return s.mux.Close()
}
