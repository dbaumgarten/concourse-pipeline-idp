package main

import (
	"context"
	"log"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/controller"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/keys"
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
	}

	jwk, kid, public, private, err := keys.GenerateJWK()
	if err != nil {
		panic(err)
	}
	server := keys.NewJWKSServer(jwk)

	gen := token.Generator{
		Issuer:          "http://localhost",
		SingingKey:      private,
		VerificationKey: public,
		KeyID:           kid,
		JWKSURL:         "https://localhost/keys",
		TTL:             30 * time.Second,
		Audiences:       []string{"sts.amazonaws.com"},
	}

	ctx := context.Background()

	out := &storage.Dummy{}

	for _, p := range pipelines {
		ctl := controller.Controller{
			Pipeline:       p,
			TokenGenerator: gen,
			Storage:        out,
			RenewBefore:    11 * time.Second,
		}
		log.Printf("Starting controller for pipeline %s", p)
		go ctl.Run(ctx)
	}

	server.ListenAndServe(":8080")
}
