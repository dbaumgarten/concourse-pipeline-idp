package storage

import (
	"context"
	"errors"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

var ErrTokenNotFound = errors.New("No stored token found for pipeline")

type Storage interface {
	ReadToken(ctx context.Context, p concourse.Pipeline) (string, error)
	WriteToken(ctx context.Context, p concourse.Pipeline, token string) error

	StoreKey(ctx context.Context, key jwk.Key) error
	GetKeys(ctx context.Context) (jwk.Set, error)
}
