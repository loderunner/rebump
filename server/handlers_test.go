package server

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"

	"github.com/loderunner/rebump/api"
)

var srv *Server

const timeout = 45 * time.Second

func TestMain(m *testing.M) {
	setUp()        // Setup for tests
	res := m.Run() // Run the actual tests
	tearDown()     // Teardown after running the tests
	os.Exit(res)
}

func setUp() {
	srv = new(Server)
	var err error
	srv.Tile38, err = redis.Dial("tcp", "localhost:9851",
		redis.DialConnectTimeout(timeout),
		redis.DialReadTimeout(timeout),
		redis.DialWriteTimeout(timeout),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to Tile38 server: %s", err))
	}
}

func tearDown() {
	srv.Tile38.Close()
}

func TestCreateBump(t *testing.T) {

	req := &api.CreateBumpRequest{Location: &api.Location{Latitude: 1435.0, Longitude: 234.0}}
	res, err := srv.CreateBump(context.Background(), req)
	if err != nil {
		t.Errorf("expected %s, got error \"%s\"", req.Location, err)
	} else if !proto.Equal(res.Location, req.Location) {
		t.Errorf("expected %s, got %s", req.Location, res.Location)
	}
}
