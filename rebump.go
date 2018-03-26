package main

import (
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/loderunner/rebump/server"
	log "github.com/sirupsen/logrus"
)

const timeout = 45 * time.Second

func main() {
	log.SetOutput(os.Stdout)

	grpcAddress := ":8080"
	restAddress := ":8081"

	srv := new(server.Server)

	// Connect to Tile38 server
	var err error
	srv.Tile38, err = redis.Dial("tcp", "localhost:9851",
		redis.DialConnectTimeout(timeout),
		redis.DialReadTimeout(timeout),
		redis.DialWriteTimeout(timeout),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Tile38 server: %s", err)
	}

	go func() {
		log.Infof("Serving REST API on \"%s\"", restAddress)
		err := srv.ListenAndServeREST(restAddress, grpcAddress)
		log.Fatalf("Failed to serve REST API: %s", err)
	}()

	log.Infof("Serving gRPC API on \"%s\"", grpcAddress)
	err = srv.ListenAndServeGRPC(grpcAddress)
	log.Fatalf("Failed to serve gRPC API: %s", err)
}
