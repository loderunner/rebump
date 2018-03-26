package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/loderunner/rebump/api"
)

const (
	ResourceNameBump = "bump"
)

func base64UUID(id *uuid.UUID) string { return base64.RawURLEncoding.EncodeToString(id[:]) }
func hexUUID(id *uuid.UUID) string    { return hex.EncodeToString(id[:]) }

type couchDBError struct {
	Error  string
	Reason string
}

func (srv *Server) CreateBump(ctx context.Context, req *api.CreateBumpRequest) (res *api.Bump, err error) {

	id := uuid.New()
	key := uuid.New()

	res = &api.Bump{
		Name:     path.Join(ResourceNameBump, base64UUID(&id)),
		Location: req.Location,
		Secret:   &api.Bump_Secret{Key: base64UUID(&key)},
	}

	// Insert bump into CouchDB
	buf, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Failed to write Bump to JSON: %s", err)
		return nil, status.Error(codes.Internal, "failed to create bump")
	}

	url := "http://localhost:5984/bump/" + hexUUID(&id)
	couchReq, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(buf))
	if err != nil {
		log.Errorf("Failed to prepare CouchDB request: %s", err)
		return nil, status.Error(codes.Internal, "failed to create bump")
	}
	couchRes, err := http.DefaultClient.Do(couchReq)
	if err != nil {
		log.Errorf("Failed to insert Bump into CouchDB: %s", err)
		return nil, status.Error(codes.Internal, "failed to create bump")
	}
	buf, err = ioutil.ReadAll(couchRes.Body)
	couchRes.Body.Close()
	if couchRes.StatusCode/100 != 2 {
		// Status code is not in the 2xx Success range
		var couchErr couchDBError
		if err := json.Unmarshal(buf, &couchErr); err != nil {
			log.Errorf("Failed to insert Bump to CouchDB: %s", couchErr.Reason)
		} else {
			log.Errorf("Failed to insert Bump to CouchDB")
		}
		switch couchRes.StatusCode {
		case http.StatusBadRequest:
			return nil, status.Error(codes.InvalidArgument, "failed to create bump: invalid argument")
		case http.StatusUnauthorized:
			return nil, status.Error(codes.PermissionDenied, "failed to create bump: permission denied")
		case http.StatusNotFound:
			return nil, status.Error(codes.NotFound, "failed to create bump: not found")
		case http.StatusConflict:
			return nil, status.Error(codes.FailedPrecondition, "failed to create bump: conflict")
		default:
			return nil, status.Error(codes.Unknown, "failed to create bump")
		}
	}

	// Insert bump location into Tile38
	_, err = srv.Tile38.Do(
		"SET",
		"bump",
		base64UUID(&id),
		"POINT",
		res.Location.Latitude,
		res.Location.Longitude,
	)
	if err != nil {
		log.Errorf("Failed to insert Bump location to Tile38: %s", err)
		return nil, status.Error(codes.Unknown, "failed to create bump")
	}

	return res, nil
}
