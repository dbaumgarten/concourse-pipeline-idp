package internal

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"github.com/go-jose/go-jose/v4"
)

type KeyManager struct {
	Storage           Storage
	TokenGenerator    *TokenGenerator
	KeyRotationPeriod time.Duration
	KeyMaxAge         time.Duration
}

func (m KeyManager) Manage(ctx context.Context) error {
	for {
		next, err := m.ManageOnce(ctx)
		if err != nil {
			log.Println(err)
		}
		delay := next.Sub(time.Now()) + 2*time.Second
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

func (m KeyManager) ManageOnce(ctx context.Context) (time.Time, error) {

	log.Print("Checking keys")
	errRetryTime := time.Now().Add(10 * time.Minute)

	currentKeys, existing, err := LoadOrGenerateAndStoreKeys(ctx, m.Storage)
	if err != nil {
		return errRetryTime, err
	}

	newestKey := findNewestKey(currentKeys)
	newestKeyCreatedAt, err := getKeyCreationTime(*newestKey)
	if err != nil {
		return errRetryTime, err
	}

	if !existing {
		log.Println("No existing signing keys found. Generated new key:", newestKey.KeyID)
	}

	if m.TokenGenerator != nil {
		m.TokenGenerator.SetKey(*newestKey)
	}

	keysChanged := false
	var nextRun time.Time

	if time.Now().Sub(newestKeyCreatedAt) > m.KeyRotationPeriod {
		log.Println("Generating new signing key")

		newKey, err := GenerateNewKey()
		if err != nil {
			return errRetryTime, err
		}

		currentKeys.Keys = append(currentKeys.Keys, *newKey)
		newestKey = newKey
		keysChanged = true
		nextRun = time.Now().Add(m.KeyRotationPeriod)
	} else {
		nextRun = newestKeyCreatedAt.Add(m.KeyRotationPeriod)
	}

	currentKeys.Keys = slices.DeleteFunc(currentKeys.Keys, func(key jose.JSONWebKey) bool {
		createdAt, err := getKeyCreationTime(key)
		if err != nil {
			return false
		}
		if time.Now().Sub(createdAt) > m.KeyMaxAge {
			log.Println("Deleting outdated signing key", key.KeyID)
			keysChanged = true
			return true
		} else {
			deletionSheduledFor := createdAt.Add(m.KeyMaxAge)
			if deletionSheduledFor.Before(nextRun) {
				nextRun = deletionSheduledFor
			}
		}
		return false
	})

	if keysChanged {
		err = m.Storage.StoreKeys(ctx, currentKeys)
		if err != nil {
			return errRetryTime, err
		}
		if m.TokenGenerator != nil {
			m.TokenGenerator.SetKey(*newestKey)
		}
	}

	return nextRun, nil
}

func LoadOrGenerateAndStoreKeys(ctx context.Context, store Storage) (jose.JSONWebKeySet, bool, error) {
	signingKeys, err := store.GetKeys(ctx)
	if err != nil && err != ErrNoKeysFound {
		return jose.JSONWebKeySet{}, false, fmt.Errorf("error when trying to fetch existing keys: %w", err)
	}

	if len(signingKeys.Keys) == 0 {
		key, err := GenerateNewKey()
		if err != nil {
			return jose.JSONWebKeySet{}, false, fmt.Errorf("error when trying to generate new key: %w", err)
		}
		signingKeys.Keys = append(signingKeys.Keys, *key)
		err = store.StoreKeys(ctx, signingKeys)
		if err != nil {
			return jose.JSONWebKeySet{}, false, fmt.Errorf("error when trying to store newly generated key: %w", err)
		}
		return signingKeys, false, nil
	}

	return signingKeys, true, nil
}

func GenerateNewKey() (*jose.JSONWebKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	return &jose.JSONWebKey{
		KeyID:     generateKID(),
		Algorithm: "RS256",
		Key:       privateKey,
		Use:       "sign",
	}, nil
}

func generateKID() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func getKeyCreationTime(key jose.JSONWebKey) (time.Time, error) {
	newestKeyUnixTime, err := strconv.ParseInt(key.KeyID, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(newestKeyUnixTime, 0), nil
}

func findNewestKey(jwks jose.JSONWebKeySet) *jose.JSONWebKey {
	var newestKey *jose.JSONWebKey
	var highestKeyID string

	for _, jwk := range jwks.Keys {
		key := jwk
		if highestKeyID == "" || key.KeyID > highestKeyID {
			newestKey = &key
			highestKeyID = key.KeyID
		}
	}

	return newestKey
}
