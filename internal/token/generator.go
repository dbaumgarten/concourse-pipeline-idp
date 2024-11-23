package token

import (
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/pipeline"
	"github.com/golang-jwt/jwt/v5"
)

type Generator struct {
	Issuer     string
	SingingKey interface{}
	Audiences  []string
	TTL        time.Duration
}

func (g Generator) Generate(p pipeline.ConcoursePipeline) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":      g.Issuer,
		"aud":      g.Audiences,
		"sub":      p.String(),
		"team":     p.Team,
		"pipeline": p.Name,
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"exp":      now.Add(g.TTL).Unix(),
		"jti":      generateJTI(),
	},
	)

	return token.SignedString(g.SingingKey)
}

func (g Generator) IsTokenStillValid(token string) (bool, time.Duration, error) {
	parser := jwt.NewParser(jwt.WithIssuer(g.Issuer), jwt.WithExpirationRequired())
	parsed, err := parser.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return g.SingingKey, nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return false, 0, nil
		}
		return false, 0, err
	}

	exp, err := parsed.Claims.GetExpirationTime()
	if err != nil {
		return false, 0, err
	}

	return parsed.Valid, time.Until(exp.Time), err
}

func generateJTI() string {
	num, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		// should never happen
		panic(err)
	}
	return strconv.Itoa(int(num.Int64()))
}
