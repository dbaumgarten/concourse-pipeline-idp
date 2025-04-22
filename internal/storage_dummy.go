package internal

import (
	"context"
	"log"
	"time"

	"github.com/go-jose/go-jose/v4"
)

type Dummy struct {
	tokens map[string]string
	jwks   jose.JSONWebKeySet
}

func (o *Dummy) WriteToken(_ context.Context, t TokenConfig, token string) error {

	if o.tokens == nil {
		o.tokens = make(map[string]string)
	}
	o.tokens[t.String()] = token

	log.Printf("Received new token: %s", token)
	return nil
}

func (o *Dummy) ReadToken(_ context.Context, t TokenConfig) (string, error) {
	if o.tokens != nil {
		if token, exists := o.tokens[t.String()]; exists {
			return token, nil
		}
	}
	return "", ErrTokenNotFound
}

func (o *Dummy) StoreKey(ctx context.Context, key jose.JSONWebKey) error {
	if o.jwks.Keys == nil {
		o.jwks = jose.JSONWebKeySet{
			Keys: make([]jose.JSONWebKey, 1),
		}
	}
	o.jwks.Keys[0] = key
	return nil
}

func (o *Dummy) GetKeys(ctx context.Context) (jose.JSONWebKeySet, error) {
	if o.jwks.Keys == nil {
		return jose.JSONWebKeySet{}, nil
	}
	return o.jwks, nil
}

func (o *Dummy) Lock(ctx context.Context, name string, duration time.Duration) error {
	return nil
}

func (o *Dummy) ReleaseLock(ctx context.Context) error {
	return nil
}
