package service

import (
	"net"
	"net/rpc"
	"os"
	"time"

	"github.com/hashicorp/raft"
	"github.com/icexin/raftkv/config"
	"github.com/icexin/raftkv/proto"
)

type Server struct {
	cfg  *config.Config
	raft *raft.Raft
	fsm  *FSM
	kvs  *KVS
	mux  *proto.Mux
}

func NewServer(cfg *config.Config) (*Server, error) {
	server := &Server{
		cfg: cfg,
	}

	// setup listener
	l, err := net.Listen("tcp", cfg.Server.Listen)
	if err != nil {
		return nil, err
	}

	// setup mux
	mux := proto.NewMux(l, nil)
	raftl := mux.Handle(proto.RaftProto)
	rpcl := mux.Handle(proto.RpcProto)

	// setup rpc server
	kvs := NewKVS(server)
	err = rpc.RegisterName("KVS", kvs)
	if err != nil {
		return nil, err
	}
	go rpc.Accept(rpcl)

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

	server.raft = raft
	server.fsm = fsm
	server.kvs = kvs
	server.mux = mux
	return server, nil
}

func (s *Server) forward(method string, req, rep interface{}) (bool, error) {
	return false, nil
}

func (s *Server) Serve() error {
	return s.mux.Serve()
}
