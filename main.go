package main

import (
	"flag"
	"log"

	_ "expvar"
	_ "net/http/pprof"

	"github.com/BurntSushi/toml"
	"github.com/icexin/raftkv/config"
	"github.com/icexin/raftkv/service"
)

var (
	cfgpath = flag.String("config", "cfg.toml", "config file path")
)

func main() {
	flag.Parse()

	var cfg config.Config
	_, err := toml.DecodeFile(*cfgpath, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	server, err := service.NewServer(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.Serve())
}
