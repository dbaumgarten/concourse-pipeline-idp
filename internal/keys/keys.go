package keys

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

func LoadOrGenerateAndStoreKeys(ctx context.Context, store storage.Storage) (jwk.Set, bool, error) {
	signingKeys, err := store.GetKeys(ctx)
	if err != nil && err != storage.ErrNoKeysFound {
		return nil, false, fmt.Errorf("error when trying to fetch existing keys: %w", err)
	}

	if signingKeys.Len() == 0 {
		log.Println("No existing keys found! Generating new one")
		key, err := GenerateNewKey()
		if err != nil {
			return nil, false, fmt.Errorf("error when trying to generate new key: %w", err)
		}
		signingKeys.AddKey(key)
		err = store.StoreKey(ctx, key)
		if err != nil {
			return nil, false, fmt.Errorf("error when trying to store newly generated key: %w", err)
		}
		return signingKeys, false, nil
	}

	return signingKeys, true, nil
}

func GenerateNewKey() (jwk.Key, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	key, err := jwk.Import(privateKey)
	if err != nil {
		return nil, err
	}

	key.Set("kid", generateKID())
	key.Set("iat", time.Now().Unix())

	return key, nil
}

func generateKID() string {
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		// should never happen
		panic(err)
	}
	return strconv.Itoa(int(num.Int64()))
}

func FindNewestKey(keys jwk.Set) jwk.Key {
	var newestKey jwk.Key
	var newestKeyCreatedAt time.Time

	for i := 0; i < keys.Len(); i++ {
		key, _ := keys.Key(i)
		if newestKey == nil {
			newestKey = key
			continue
		}
		var createdAt time.Time
		key.Get("iat", &createdAt)
		if createdAt.After(newestKeyCreatedAt) {
			newestKey = key
			newestKeyCreatedAt = createdAt
		}
	}

	return newestKey
}
