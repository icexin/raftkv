package service

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/hashicorp/raft"
	"github.com/icexin/raftkv/config"
	"github.com/icexin/raftkv/proto"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	errBadMethod = errors.New("bad method")
	errBadAction = errors.New("bad action")
)

type FSM struct {
	cfg *config.DB
	*leveldb.DB
}

func NewFSM(cfg *config.DB) (*FSM, error) {
	db, err := leveldb.OpenFile(cfg.Dir, nil)
	if err != nil {
		return nil, err
	}

	return &FSM{
		cfg: cfg,
		DB:  db,
	}, nil
}

func (f *FSM) Apply(l *raft.Log) interface{} {
	req := new(proto.Request)
	err := proto.Unmarshal(l.Data, req)
	if err != nil {
		return err
	}
	switch req.Action {
	case proto.OpWrite:
		return f.Put(req.Key, req.Data, nil)
	case proto.OpDelete:
		return f.Delete(req.Key, nil)
	default:
		return errBadAction
	}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	snapshot, err := f.GetSnapshot()
	if err != nil {
		return nil, err
	}
	return &fsmSnapshot{snapshot}, nil
}

func (f *FSM) Restore(r io.ReadCloser) error {
	defer r.Close()

	zr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	oldname := f.cfg.Dir + ".old"
	err = os.Rename(f.cfg.Dir, oldname)
	if err != nil {
		return err
	}
	defer os.RemoveAll(oldname)

	tr := tar.NewReader(zr)
	err = Untar(f.cfg.Dir, tr)
	if err != nil {
		return err
	}

	db, err := leveldb.OpenFile(f.cfg.Dir, nil)
	if err != nil {
		return err
	}

	f.DB = db
	return nil
}

// fsmSnapshot implement FSMSnapshot interface
type fsmSnapshot struct {
	snapshot *leveldb.Snapshot
}

// At first, walk all kvs, write temp leveldb.
// Second, make tar.gz for temp leveldb dir
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	// Create a temporary path for the state store
	tmpPath, err := ioutil.TempDir(os.TempDir(), "state")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpPath)

	db, err := leveldb.OpenFile(tmpPath, nil)
	if err != nil {
		return err
	}
	iter := f.snapshot.NewIterator(nil, nil)
	for iter.Next() {
		err = db.Put(iter.Key(), iter.Value(), nil)
		if err != nil {
			db.Close()
			sink.Cancel()
			return err
		}
	}
	iter.Release()
	db.Close()

	// make tar.gz
	w := gzip.NewWriter(sink)
	err = Tar(tmpPath, w)
	if err != nil {
		sink.Cancel()
		return err
	}

	err = w.Close()
	if err != nil {
		sink.Cancel()
		return err
	}

	sink.Close()
	return nil
}

func (f *fsmSnapshot) Release() {
	f.snapshot.Release()
}
