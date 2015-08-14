package config

import "time"

type Config struct {
	Raft   Raft
	Server Server
	DB     DB
}

type Server struct {
	Listen string
}

type Raft struct {
	Advertise string
	DataDir   string

	SnapshotInterval  Duration
	SnapshotThreshold uint64
	EnableSingleNode  bool
}

type DB struct {
	Dir string
}

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	du, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(du)
	return nil
}
