package keys

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"

	"github.com/MicahParks/jwkset"
)

func GenerateJWK() (jwk jwkset.JWK, kid string, public *rsa.PublicKey, private interface{}, err error) {
	kid = "abcd"

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return
	}

	private = privateKey
	public = &privateKey.PublicKey

	// Create the JWK from the key and options.
	jwk, err = jwkset.NewJWKFromKey(private, jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			KID: kid,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to create JWK from key: %w", err)
		return
	}

	return
}

type JWKSServer struct {
	JWKS jwkset.Storage
}

func NewJWKSServer(jwk jwkset.JWK) *JWKSServer {
	jwkSet := jwkset.NewMemoryStorage()

	ctx := context.Background()
	// Write the key to the JWK Set storage.
	err := jwkSet.KeyWrite(ctx, jwk)
	if err != nil {
		panic(err)
	}

	return &JWKSServer{
		JWKS: jwkSet,
	}
}

func (s JWKSServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	response, err := s.JWKS.JSONPublic(request.Context())
	if err != nil {
		log.Printf("Failed to get JWK Set JSON %s", err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write(response)
}

func (s JWKSServer) ListenAndServe(addr string) {
	http.ListenAndServe(addr, s)
}
