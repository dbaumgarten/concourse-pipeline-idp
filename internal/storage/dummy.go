package storage

import (
	"context"
	"log"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
)

type Dummy struct {
	tokens map[string]string
	keys   []interface{}
}

func (o *Dummy) WriteToken(_ context.Context, p concourse.Pipeline, token string) error {

	if o.tokens == nil {
		o.tokens = make(map[string]string)
	}
	o.tokens[p.String()] = token

	log.Printf("Received new token: %s", token)
	return nil
}

func (o *Dummy) ReadToken(_ context.Context, p concourse.Pipeline) (string, error) {
	if o.tokens != nil {
		if token, exists := o.tokens[p.String()]; exists {
			return token, nil
		}
	}
	return "", ErrTokenNotFound
}

func (o *Dummy) StoreKey(ctx context.Context, key interface{}) error {
	if o.keys == nil {
		o.keys = make([]interface{}, 0, 1)
	}
	o.keys = append(o.keys, key)
	return nil
}

func (o *Dummy) GetKeys(ctx context.Context) ([]interface{}, error) {
	if o.keys == nil {
		return make([]interface{}, 0), nil
	}
	return o.keys, nil
}
