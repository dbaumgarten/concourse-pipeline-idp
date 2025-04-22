package internal

import (
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type TokenGenerator struct {
	Issuer string
	Key    jose.JSONWebKey
}

func (g TokenGenerator) Generate(conf TokenConfig) (token string, validUntil time.Time, err error) {
	now := time.Now()
	validUntil = now.Add(conf.ExpiresIn)

	signingKey := jose.SigningKey{
		Algorithm: jose.SignatureAlgorithm(g.Key.Algorithm),
		Key:       g.Key,
	}

	signer, err := jose.NewSigner(signingKey, &jose.SignerOptions{})
	if err != nil {
		return "", time.Time{}, err
	}

	claims := jwt.Claims{
		Issuer:    g.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Audience:  jwt.Audience(conf.Audience),
		Subject:   conf.Subject(),
		Expiry:    jwt.NewNumericDate(validUntil),
		ID:        generateJTI(),
	}

	customClaims := struct {
		Team     string `json:"team"`
		Pipeline string `json:"pipeline"`
	}{
		Team:     conf.Team,
		Pipeline: conf.Pipeline,
	}

	signed, err := jwt.Signed(signer).Claims(claims).Claims(customClaims).Serialize()
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, validUntil, nil
}

func (g TokenGenerator) IsTokenStillValid(token string) (bool, time.Time, error) {

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(g.Key.Algorithm)})
	if err != nil {
		return false, time.Time{}, err
	}

	claims := jwt.Claims{}
	err = parsed.Claims(g.Key, &claims)
	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}

	return true, claims.Expiry.Time(), nil
}

func generateJTI() string {
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		// should never happen
		panic(err)
	}
	return strconv.Itoa(int(num.Int64()))
}
