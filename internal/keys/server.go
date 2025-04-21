package keys

import (
	"encoding/json"
	"net/http"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/go-jose/go-jose/v4"
)

type JWKSServer struct {
	*http.ServeMux
	store       storage.Storage
	externalURL string
}

func NewJWKSServer(store storage.Storage, externalURL string) JWKSServer {
	s := JWKSServer{
		ServeMux:    http.NewServeMux(),
		store:       store,
		externalURL: externalURL,
	}

	s.Handle("/.well-known/openid-configuration", http.HandlerFunc(s.serveDiscovery))
	s.Handle("/keys", http.HandlerFunc(s.serveKeys))

	return s
}

func (s JWKSServer) serveDiscovery(writer http.ResponseWriter, request *http.Request) {
	resp := struct {
		Issuer  string `json:"issuer"`
		JWKSUri string `json:"jwks_uri"`
	}{
		Issuer:  s.externalURL,
		JWKSUri: s.externalURL + "/keys",
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(resp)
}

func (s JWKSServer) serveKeys(writer http.ResponseWriter, request *http.Request) {
	jwks, err := s.store.GetKeys(request.Context())
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	pubKeys := jose.JSONWebKeySet{
		Keys: make([]jose.JSONWebKey, len(jwks.Keys)),
	}
	for i, key := range jwks.Keys {
		pubKey := key.Public()
		pubKeys.Keys[i] = pubKey
	}

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(pubKeys)
}

func (s JWKSServer) ListenAndServe(addr string) {
	http.ListenAndServe(addr, s)
}
