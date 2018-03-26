package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/loderunner/rebump/api"
)

type Server struct {
	Tile38 redis.Conn
}

func (s *Server) ListenAndServeGRPC(address string) error {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				logInterceptor,
			),
		),
	}
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

// unaryLogInterceptor logs the request after it passes through the handler. The log
// format is similar to the NCSA combined log format, replacing:
//  * the HTTP request line with the gRPC method name
//  * the HTTP return status code with the gRPC return code
//  * the size of the response in bytes with the size of the serialized protobuf message
// Apache's mod_log_config format string:
// "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-agent}i\""
func logInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (res interface{}, err error) {
	// Store incoming request time
	reqTime := time.Now()

	// Call the actual handler (or next interceptor in the chain)
	res, err = handler(ctx, req)

	// Initialize all strings
	addr := "-"                                                 // The remote hostname
	logname := "-"                                              // Always "-", has something to do with identd
	username := "-"                                             // Remote user if the request was authenticated.
	timestamp := reqTime.Format("[02/Jan/2006 03:04:05 -0700]") // Time the request was received
	request := info.FullMethod                                  // The full gRPC method name
	statusCode := "-"                                           // The gRPC return status
	byteSize := "-"                                             // The size of the response in bytes
	referer := "-"                                              // The Referer header, if any
	userAgent := "-"                                            // The User-Agent header, if any

	// Get the remote peer info from the context
	if p, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			addr = tcpAddr.IP.String()
		} else {
			addr = p.Addr.String()
		}
	}

	// TODO: Get the username if logged in

	if err == nil {
		statusCode = codes.OK.String()
		byteSize = strconv.Itoa(proto.Size(res.(proto.Message)))
	} else if grpcErr, ok := status.FromError(err); ok {
		statusCode = grpcErr.Code().String()
		byteSize = strconv.Itoa(proto.Size(err.(proto.Message)))
	}

	// Get metadata from context
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if r, ok := md["referer"]; ok {
			referer = strings.Join(r, "")
		}
		if ua, ok := md["user-agent"]; ok {
			userAgent = strings.Join(ua, "")
		}
	}

	var b strings.Builder
	fmt.Fprintf(
		&b,
		"%s %s %s %s \"%s\" %s %s \"%s\" \"%s\"",
		addr,
		logname,
		username,
		timestamp,
		request,
		statusCode,
		byteSize,
		referer,
		userAgent,
	)

	log.Infof("%s", b.String())

	return
}
