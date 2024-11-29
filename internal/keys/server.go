package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"math"
	"math/big"
	"net/http"
	"strconv"

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

type JWKSServer struct {
	keyset jwk.Set
}

func NewJWKSServer(keys ...jwk.Key) JWKSServer {
	set := jwk.NewSet()
	for _, key := range keys {
		pubkey, _ := key.PublicKey()
		set.AddKey(pubkey)
	}

	return JWKSServer{
		keyset: set,
	}
}

func (s JWKSServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(s.keyset)
}

func (s JWKSServer) ListenAndServe(addr string) {
	http.ListenAndServe(addr, s)
}
