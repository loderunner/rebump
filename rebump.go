package main

import (
	"fmt"
	"os"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/loderunner/rebump/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

const VERSION = "0.0.0"

const TIMEOUT = 45 * time.Second

// global variables for the flags
var (
	version        bool
	grpcAddress    string
	restAddress    string
	couchDBAddress string
	tile38Address  string
	logLevel       uint32
)

func parseFlags(args []string) {

	// Version
	pflag.BoolVarP(&version, "version", "V", false, "display version and quit")

	// gRPC  address
	pflag.StringVar(&grpcAddress, "grpc-addr", ":8080", "address to bind the gRPC server to")

	// REST address
	pflag.StringVar(&restAddress, "rest-addr", ":8081", "address to bind the REST server to")

	// CouchDB address
	pflag.StringVar(&couchDBAddress, "couchdb-addr", "127.0.0.1:5984", "address of the CouchDB server")

	// Tile38 address
	pflag.StringVar(&tile38Address, "tile38-addr", "127.0.0.1:9851", "address of the Tile38 server")

	// Verbosity
	pflag.Uint32VarP(&logLevel, "verbose", "v", uint32(log.InfoLevel), "verbosity level [0:quiet - 5:debug]")

	pflag.Parse()
}

func main() {
	parseFlags(os.Args)

	if version {
		fmt.Println("re:BUMP version", version)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(log.Level(logLevel))
	log.StandardLogger().Formatter.(*log.TextFormatter).FullTimestamp = true

	srv := new(server.Server)

	// Connect to Tile38 server
	var err error
	srv.Tile38, err = redis.Dial("tcp", tile38Address,
		redis.DialConnectTimeout(TIMEOUT),
		redis.DialReadTimeout(TIMEOUT),
		redis.DialWriteTimeout(TIMEOUT),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Tile38 server: %s", err)
	}

	srv.CouchDBAddress = couchDBAddress

	go func() {
		log.Infof("Serving REST API on \"%s\"", restAddress)
		err := srv.ListenAndServeREST(restAddress, grpcAddress)
		log.Fatalf("Failed to serve REST API: %s", err)
	}()

	log.Infof("Serving gRPC API on \"%s\"", grpcAddress)
	err = srv.ListenAndServeGRPC(grpcAddress)
	log.Fatalf("Failed to serve gRPC API: %s", err)
}
