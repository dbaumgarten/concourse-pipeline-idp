package storage

import (
	"context"
	"errors"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/pipeline"
)

var ErrTokenNotFound = errors.New("No stored token found for pipeline")

type ReadWriter interface {
	ReadToken(ctx context.Context, p pipeline.ConcoursePipeline) (string, error)
	WriteToken(ctx context.Context, p pipeline.ConcoursePipeline, token string) error
}
