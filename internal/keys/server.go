package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

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

type JWKSServer struct {
	store storage.Storage
}

func NewJWKSServer(store storage.Storage) JWKSServer {
	return JWKSServer{
		store: store,
	}
}

func (s JWKSServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	keys, err := s.store.GetKeys(request.Context())
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	pubKeys := jwk.NewSet()
	for i := 0; i < keys.Len(); i++ {
		key, _ := keys.Key(i)
		pubKey, _ := key.PublicKey()
		pubKeys.AddKey(pubKey)
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(pubKeys)
}

func (s JWKSServer) ListenAndServe(addr string) {
	http.ListenAndServe(addr, s)
}
