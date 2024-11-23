package pipeline

import "context"

type StaticList []ConcoursePipeline

func (l StaticList) List(_ context.Context) ([]ConcoursePipeline, error) {
	return l, nil
}
