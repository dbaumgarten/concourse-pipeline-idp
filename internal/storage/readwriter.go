package storage

import (
	"context"
	"errors"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
)

var ErrTokenNotFound = errors.New("No stored token found for pipeline")

type ReadWriter interface {
	ReadToken(ctx context.Context, p concourse.Pipeline) (string, error)
	WriteToken(ctx context.Context, p concourse.Pipeline, token string) error
}
