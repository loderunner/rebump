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

func fromB64(id string) (*uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return nil, err
	}
	u, err := uuid.FromBytes(b)
	return &u, err
}

func toB64(id *uuid.UUID) string { return base64.RawURLEncoding.EncodeToString(id[:]) }

func fromHex(h string) (*uuid.UUID, error) {
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, err
	}
	u, err := uuid.FromBytes(b)
	return &u, err
}

func toHex(id *uuid.UUID) string { return hex.EncodeToString(id[:]) }

func (srv *Server) CreateBump(ctx context.Context, req *api.CreateBumpRequest) (res *api.Bump, err error) {
	if req.Location == nil {
		log.Errorf("Missing location in request: %s", req)
		return nil, status.Error(codes.InvalidArgument, "missing location in request")
	}

	id := uuid.New()
	key := uuid.New()

	res = &api.Bump{
		Name:     path.Join(ResourceNameBump, toB64(&id)),
		Location: req.Location,
		Secret:   &api.Bump_Secret{Key: toB64(&key)},
	}

	// Insert bump into CouchDB
	buf, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Failed to write Bump to JSON: %s", err)
		return nil, status.Error(codes.Internal, "failed to create bump")
	}

	url := "http://localhost:5984/bump/" + toHex(&id)
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
		var couchErr map[string]interface{}
		if err := json.Unmarshal(buf, &couchErr); err != nil {
			log.Errorf("Failed to insert Bump to CouchDB: %s", couchErr["reason"])
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
		toB64(&id),
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

func (srv *Server) GetNearby(ctx context.Context, req *api.GetNearbyRequest) (res *api.Bump, err error) {
	if req.Location == nil {
		log.Errorf("Missing location in request: %s", req)
		return nil, status.Error(codes.InvalidArgument, "missing location in request")
	}

	// Get bump location from Tile38
	tile38Res, err := srv.Tile38.Do(
		"NEARBY",
		"bump",
		"LIMIT",
		1,
		"IDS",
		"POINT",
		req.Location.Latitude,
		req.Location.Longitude,
		20,
	)
	if err != nil {
		log.Errorf("Failed to retrieve nearby Bump location from Tile38: %s", err)
		return nil, status.Error(codes.Unknown, "failed to retrieve bump")
	}
	var ids []interface{}
	var ok bool
	if ids, ok = tile38Res.([]interface{}); !ok {
		log.Errorf("Failed to read ids from Tile38: %#v", tile38Res)
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}
	if len(ids) != 2 {
		log.Errorf("Failed to read ids from Tile38: %#v", tile38Res)
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}
	if ids, ok = ids[1].([]interface{}); !ok {
		log.Errorf("Failed to read ids from Tile38: %#v", tile38Res)
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}
	if len(ids) == 0 {
		log.Error("Couldn't find nearby Bump")
		return nil, status.Error(codes.NotFound, "no bump nearby")
	}
	var id []byte
	if id, ok = ids[0].([]byte); !ok {
		log.Errorf("Couldn't read Bump ID from Tile83: %#v", ids[0])
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}

	// Parse UUID
	u, err := fromB64(string(id))
	if err != nil {
		log.Errorf("Couldn't read id from base64: %s", err)
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}

	// Get Bump from CouchDB
	url := "http://localhost:5984/bump/" + toHex(u)
	couchRes, err := http.DefaultClient.Get(url)
	if err != nil {
		log.Errorf("Failed to retrieve Bump from CouchDB: %s", err)
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}
	log.Debugf("Got response from CouchDB: %s", couchRes)
	buf, err := ioutil.ReadAll(couchRes.Body)
	log.Debugf("Body: %s", string(buf))
	couchRes.Body.Close()
	if couchRes.StatusCode/100 != 2 {
		// Status code is not in the 2xx Success range
		var couchErr map[string]interface{}
		if err := json.Unmarshal(buf, &couchErr); err != nil {
			log.Errorf("Failed to retrieve Bump from CouchDB: %s", couchErr["reason"])
		} else {
			log.Errorf("Failed to retrieve Bump from CouchDB")
		}
		switch couchRes.StatusCode {
		case http.StatusUnauthorized:
			return nil, status.Error(codes.PermissionDenied, "failed to retrieve bump: permission denied")
		case http.StatusNotFound:
			return nil, status.Error(codes.NotFound, "failed to retrieve bump: not found")
		default:
			return nil, status.Error(codes.Unknown, "failed to retrieve bump")
		}
	}
	res = new(api.Bump)
	err = json.Unmarshal(buf, res)
	if err != nil {
		log.Errorf("Failed to unmarshal JSON from CouchDB: %s", err)
		return nil, status.Error(codes.Internal, "failed to retrieve bump")
	}

	// Remove secret
	res.Secret = nil

	return res, nil
}
