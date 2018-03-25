package server

import (
	"context"
	"encoding/base64"
	"path"

	"github.com/google/uuid"

	"github.com/loderunner/rebump/api"
)

const (
	ResourceNameBump = "bump"
)

type Server struct{}

func base64UUID(id *uuid.UUID) string { return base64.RawURLEncoding.EncodeToString(id[:]) }

func (srv *Server) CreateBump(ctx context.Context, req *api.CreateBumpRequest) (res *api.Bump, err error) {

	id := uuid.New()
	key := uuid.New()

	res = &api.Bump{
		Name:     path.Join(ResourceNameBump, base64UUID(&id)),
		Location: req.Location,
		Secret:   &api.Bump_Secret{Key: base64UUID(&key)},
	}

	return res, nil
}
