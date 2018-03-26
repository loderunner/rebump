package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"github.com/loderunner/rebump/api"
)

type Server struct {
	Tile38 redis.Conn
}

func (s *Server) ListenAndServeGRPC(address string) error {
	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	api.RegisterRebumpServer(grpcServer, s)

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s : %v", address, err)
	}

	return grpcServer.Serve(lis)
}

func headerMatcher(headerName string) (string, bool) {
	return strings.ToLower(headerName), true
}

func (s *Server) ListenAndServeREST(restAddress, grpcAddress string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(runtime.WithIncomingHeaderMatcher(runtime.DefaultHeaderMatcher))

	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := api.RegisterRebumpHandlerFromEndpoint(
		ctx,
		mux,
		grpcAddress,
		opts,
	)
	if err != nil {
		return fmt.Errorf("could not register REST service: %s", err)
	}

	return http.ListenAndServe(restAddress, mux)
}
