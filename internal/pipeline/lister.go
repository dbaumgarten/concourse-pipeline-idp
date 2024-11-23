package pipeline

import "context"

type Lister interface {
	List(ctx context.Context) ([]ConcoursePipeline, error)
}
