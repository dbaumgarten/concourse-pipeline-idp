package storage

import (
	"context"
	"log"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
)

type Dummy struct {
	tokens map[string]string
}

func (o *Dummy) WriteToken(_ context.Context, p concourse.Pipeline, token string) error {

	if o.tokens == nil {
		o.tokens = make(map[string]string)
	}
	o.tokens[p.String()] = token

	log.Printf("Received new token: %s", token)
	return nil
}

func (o Dummy) ReadToken(_ context.Context, p concourse.Pipeline) (string, error) {
	if o.tokens != nil {
		if token, exists := o.tokens[p.String()]; exists {
			return token, nil
		}
	}
	return "", ErrTokenNotFound
}
