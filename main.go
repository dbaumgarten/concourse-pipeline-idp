package main

import (
	"context"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/keys"
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

	kid := "abcd"
	public, private, _ := keys.GenerateKey()
	keyset, _ := keys.CreateKeyset(kid, private)
	handler := keys.CreateHTTPHandler(keyset)
	go keys.ListenAndServe(":8080", handler)

	gen := token.Generator{
		Issuer:          "http://localhost",
		SingingKey:      private,
		VerificationKey: public,
		KeyID:           kid,
		JWKSURL:         "https://localhost/keys",
		TTL:             30 * time.Second,
		Audiences:       []string{"sts.amazonaws.com"},
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
