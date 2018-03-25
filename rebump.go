package main

import (
	"os"

	"github.com/loderunner/rebump/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetOutput(os.Stdout)

	grpcAddress := ":8080"
	restAddress := ":8081"

	go func() {
		log.Infof("Serving REST API on \"%s\"", restAddress)
		err := server.ListenAndServeREST(restAddress, grpcAddress)
		log.Fatalf("Failed to serve REST API: %s", err)
	}()

	log.Infof("Serving gRPC API on \"%s\"", grpcAddress)
	err := server.ListenAndServeGRPC(grpcAddress)
	log.Fatalf("Failed to serve gRPC API: %s", err)
}
