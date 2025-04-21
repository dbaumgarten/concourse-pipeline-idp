package storage

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/config"
	"github.com/go-jose/go-jose/v4"
)

var ErrTokenNotFound = errors.New("no stored token found for pipeline")
var ErrNoKeysFound = errors.New("could not find existing signing keys")

type Storage interface {
	ReadToken(ctx context.Context, t config.TokenConfig) (string, error)
	WriteToken(ctx context.Context, t config.TokenConfig, token string) error

	StoreKey(ctx context.Context, key jose.JSONWebKey) error
	GetKeys(ctx context.Context) (jose.JSONWebKeySet, error)

	Lock(ctx context.Context, name string, duration time.Duration) error
	ReleaseLock(ctx context.Context) error
}

// LockAndHold tries to aquire the lock of the backend. Blocks until is has the lock.
// Continues to renew the lock in the background. If renewal fails, program terminates
func LockAndHold(ctx context.Context, backend Storage, name string, ttl time.Duration, renewBefore time.Duration) error {
	err := backend.Lock(ctx, name, ttl)
	if err != nil {
		return nil
	}

	go func() {
		for {
			duration := ttl - renewBefore
			log.Printf("Renewing leader lock in %s", duration.String())
			time.Sleep(duration)
			err := backend.Lock(ctx, name, ttl)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Renewed leader lock")
		}
	}()

	return nil
}
