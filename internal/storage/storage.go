package storage

import (
	"context"
	"errors"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
)

var ErrTokenNotFound = errors.New("No stored token found for pipeline")

type Storage interface {
	ReadToken(ctx context.Context, p concourse.Pipeline) (string, error)
	WriteToken(ctx context.Context, p concourse.Pipeline, token string) error

	StoreKey(ctx context.Context, key interface{}) error
	GetKeys(ctx context.Context) ([]interface{}, error)
}
