package keys

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"strconv"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/go-jose/go-jose/v4"
)

func LoadOrGenerateAndStoreKeys(ctx context.Context, store storage.Storage) (jose.JSONWebKeySet, bool, error) {
	signingKeys, err := store.GetKeys(ctx)
	if err != nil && err != storage.ErrNoKeysFound {
		return jose.JSONWebKeySet{}, false, fmt.Errorf("error when trying to fetch existing keys: %w", err)
	}

	if len(signingKeys.Keys) == 0 {
		key, err := GenerateNewKey()
		if err != nil {
			return jose.JSONWebKeySet{}, false, fmt.Errorf("error when trying to generate new key: %w", err)
		}
		signingKeys.Keys = append(signingKeys.Keys, *key)
		err = store.StoreKey(ctx, *key)
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

func FindNewestKey(jwks jose.JSONWebKeySet) *jose.JSONWebKey {
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
