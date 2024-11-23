package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/manager"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/pipeline"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/token"
)

func main() {

	pipelines := pipeline.StaticList{
		{
			Team: "main",
			Name: "foo",
		},
		{
			Team: "other",
			Name: "bar",
		},
	}

	pkey := generateKey()

	gen := token.Generator{
		Issuer:     "http://localhost",
		SingingKey: pkey,
		TTL:        30 * time.Second,
		Audiences:  []string{"sts.amazonaws.com"},
	}

	out := &storage.Dummy{}

	mgr := manager.Manager{
		Interval:       10 * time.Second,
		Pipelines:      pipelines,
		TokenGenerator: gen,
		Storage:        out,
		RenewBefore:    11 * time.Second,
	}

	mgr.ManageLoop(context.Background())
}

func generateKey() *rsa.PrivateKey {

	bitSize := 1024

	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		panic(err)
	}

	// Encode public key to PKCS#1 ASN.1 PEM.
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(key.Public().(*rsa.PublicKey)),
		},
	)

	fmt.Println(string(pubPEM))

	return key
}
