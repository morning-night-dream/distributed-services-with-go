package main

import (
	"net"
	"os"

	"github.com/morning-night-dream/distributed-services-with-go/internal/log"
	"github.com/morning-night-dream/distributed-services-with-go/internal/server"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	go func() {
		err := srv.ListenAndServe()
		handleError(err)
	}()

	err := os.Mkdir("server-temp", 0700)
	handleError(err)

	clog, err := log.NewLog("server-temp", log.Config{})
	handleError(err)

	cfg := &server.Config{
		CommitLog: clog,
	}

	server, err := server.NewGRPCServer(cfg)
	handleError(err)

	l, err := net.Listen("tcp", ":8081")
	handleError(err)

	server.Serve(l)
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
