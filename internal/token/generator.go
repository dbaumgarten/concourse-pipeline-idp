package token

import (
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type Generator struct {
	Issuer    string
	Key       jwk.Key
	Audiences []string
	TTL       time.Duration
}

func (g Generator) Generate(p concourse.Pipeline) (token string, validUntil time.Time, err error) {
	now := time.Now()
	validUntil = now.Add(g.TTL)

	unsigned, err := jwt.NewBuilder().
		Issuer(g.Issuer).
		IssuedAt(now).
		NotBefore(now).
		Audience(g.Audiences).
		Subject(p.String()).
		Expiration(validUntil).
		JwtID(generateJTI()).
		Claim("team", p.Team).
		Claim("pipeline", p.Name).
		Build()

	if err != nil {
		return "", time.Time{}, err
	}

	signed, err := jwt.Sign(unsigned, jwt.WithKey(jwa.RS256(), g.Key))
	if err != nil {
		return "", time.Time{}, err
	}

	return string(signed), validUntil, nil
}

func (g Generator) IsTokenStillValid(token string) (bool, time.Time, error) {

	parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.RS256(), g.Key))
	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}

	exp, exists := parsed.Expiration()
	if !exists {
		return false, time.Time{}, err
	}

	return true, exp, nil
}

func generateJTI() string {
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		// should never happen
		panic(err)
	}
	return strconv.Itoa(int(num.Int64()))
}
