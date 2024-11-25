package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/config"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/controller"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/keys"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/token"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config: ", err)
		os.Exit(1)
	}

	err = cfg.Validate()
	if err != nil {
		fmt.Println("Config is invalid: ", err)
		os.Exit(1)
	}

	jwk, kid, public, private, err := keys.GenerateJWK()
	if err != nil {
		panic(err)
	}
	server := keys.NewJWKSServer(jwk)

	gen := token.Generator{
		Issuer:          cfg.ExternalURL,
		SingingKey:      private,
		VerificationKey: public,
		KeyID:           kid,
		JWKSURL:         cfg.ExternalURL + "/keys",
		TTL:             cfg.TokenOpts.TTL,
		Audiences:       cfg.TokenOpts.Audiences,
	}

	ctx := context.Background()

	out := &storage.Dummy{}

	for _, p := range cfg.Pipelines {
		ctl := controller.Controller{
			Pipeline:       p,
			TokenGenerator: gen,
			Storage:        out,
			RenewBefore:    cfg.TokenOpts.RenewBefore,
		}
		log.Printf("Starting controller for pipeline %s", p)
		go ctl.Run(ctx)
	}

	server.ListenAndServe(":8080")
}
