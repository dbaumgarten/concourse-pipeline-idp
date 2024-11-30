package keys

import (
	"encoding/json"
	"net/http"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

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
