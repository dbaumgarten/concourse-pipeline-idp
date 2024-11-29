package concourse

import "context"

type StaticList []Pipeline

func (l StaticList) List(_ context.Context) ([]Pipeline, error) {
	return l, nil
}
