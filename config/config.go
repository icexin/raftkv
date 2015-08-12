package config

type Config struct {
	Raft   Raft
	Server Server
	DB     DB
}

type Server struct {
	Listen string
}

type Raft struct {
	Listen    string
	Advertise string
	DataDir   string
}

type DB struct {
	Dir string
}
