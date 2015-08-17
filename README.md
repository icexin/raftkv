raftkv
======

# Features

* High avaiable using [raft](http://raftconsensus.github.io/)
* Redis protocol compatible

# Install

`go get github.com/icexin/raftkv`

# Getting started

Start server on port 10000

`raftkv -config=cfg.toml`

Using `redis-cli -p 10000`

# Supported commands

`GET`, `SET`, `DEL`
