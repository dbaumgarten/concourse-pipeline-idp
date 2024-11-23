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

func GenerateKey() (public, private interface{}, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	return key.Public(), key, nil
}

func CreateKeyset(kid string, key interface{}) (jwkset.Storage, error) {
	jwkSet := jwkset.NewMemoryStorage()

	options := jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			KID: kid,
		},
	}

	// Create the JWK from the key and options.
	jwk, err := jwkset.NewJWKFromKey(key, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWK from key: %w", err)
	}

	ctx := context.Background()
	// Write the key to the JWK Set storage.
	err = jwkSet.KeyWrite(ctx, jwk)
	if err != nil {
		return nil, fmt.Errorf("failed to store RSA key: %w", err)
	}

	return jwkSet, nil
}

func CreateHTTPHandler(jwkSet jwkset.Storage) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// TODO Cache the JWK Set so storage isn't called for every request.
		response, err := jwkSet.JSONPublic(request.Context())
		if err != nil {
			log.Printf("Failed to get JWK Set JSON %s", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(response)
	})
}

func ListenAndServe(addr string, keysetHandler http.HandlerFunc) {
	http.ListenAndServe(addr, keysetHandler)
}
