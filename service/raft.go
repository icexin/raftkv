package service

import (
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	"github.com/icexin/raft-leveldb"
	"github.com/icexin/raftkv/config"
	"github.com/icexin/raftkv/proto"
)

type raftLayer struct {
	advertise net.Addr
	listener  net.Listener
}

func NewRaftLayer(advertise net.Addr, l net.Listener) *raftLayer {
	return &raftLayer{
		advertise: advertise,
		listener:  l,
	}
}

func (t *raftLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write([]byte{proto.RaftProto})
	if err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// Accept implements the net.Listener interface.
func (t *raftLayer) Accept() (c net.Conn, err error) {
	return t.listener.Accept()
}

// Close implements the net.Listener interface.
func (t *raftLayer) Close() (err error) {
	return t.listener.Close()
}

// Addr implements the net.Listener interface.
func (t *raftLayer) Addr() net.Addr {
	// Use an advertise addr if provided
	if t.advertise != nil {
		return t.advertise
	}
	return t.listener.Addr()
}

func NewRaft(cfg *config.Raft, fsm raft.FSM, trans raft.Transport) (*raft.Raft, error) {
	raftLogDir := filepath.Join(cfg.DataDir, "log")
	raftMetaDir := filepath.Join(cfg.DataDir, "meta")

	logStore, err := raftleveldb.NewStore(raftLogDir)
	if err != nil {
		return nil, err
	}

	metaStore, err := raftleveldb.NewStore(raftMetaDir)
	if err != nil {
		return nil, err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(cfg.DataDir, 3, os.Stderr)
	if err != nil {
		return nil, err
	}

	peerStore := raft.NewJSONPeers(cfg.DataDir, trans)

	raftConfig := raft.DefaultConfig()
	raftConfig.SnapshotInterval = time.Second * 10
	raftConfig.EnableSingleNode = true
	return raft.NewRaft(
		raftConfig,
		fsm,
		logStore,
		metaStore,
		snapshotStore,
		peerStore,
		trans,
	)
}
