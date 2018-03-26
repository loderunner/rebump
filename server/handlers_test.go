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
	srv.CouchDBAddress = "localhost:5984"
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

func TestGetBumpNearby(t *testing.T) {
	loc := &api.Location{Latitude: 3.8, Longitude: -89.4}

	{
		req := &api.CreateBumpRequest{Location: loc}
		_, err := srv.CreateBump(context.Background(), req)
		if err != nil {
			t.Errorf("failed to create bump: %s", err)
		}
	}

	req := &api.GetBumpNearbyRequest{Location: &api.Location{Latitude: 3.8, Longitude: -89.4}}
	res, err := srv.GetBumpNearby(context.Background(), req)
	if err != nil {
		t.Errorf("expected %s, got error \"%s\"", loc, err)
	} else if !proto.Equal(res.Location, loc) {
		t.Errorf("expected %s, got %s", loc, res.Location)
	}

	req = &api.GetBumpNearbyRequest{Location: &api.Location{Latitude: 3.80001, Longitude: -89.4}}
	res, err = srv.GetBumpNearby(context.Background(), req)
	if err != nil {
		t.Errorf("expected %s, got error \"%s\"", loc, err)
	} else if !proto.Equal(res.Location, loc) {
		t.Errorf("expected %s, got %s", loc, res.Location)
	}

	req = &api.GetBumpNearbyRequest{Location: &api.Location{Latitude: -3.80001, Longitude: -89.4}}
	res, err = srv.GetBumpNearby(context.Background(), req)
	if err == nil {
		t.Errorf("expected error, got %s", res)
	}
}
