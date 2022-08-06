package main

import (
	"log"

	"github.com/morning-night-dream/distributed-services-with-go/internal/server"
)

func main() {
	srv := server.NewHTTPServer(":8080")
	log.Fatal(srv.ListenAndServe())
}
