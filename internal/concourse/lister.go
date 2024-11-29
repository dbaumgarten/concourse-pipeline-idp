package concourse

import "context"

type PipelineLister interface {
	ListPipelines(ctx context.Context) ([]Pipeline, error)
}
