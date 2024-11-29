package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/config"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/controller"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/keys"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/token"
	"github.com/hashicorp/vault-client-go"
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

	if cfg.ListenAddr != "" {
		server := keys.NewJWKSServer(jwk)
		go server.ListenAndServe(cfg.ListenAddr)
	}

	gen := &token.Generator{
		Issuer:          cfg.ExternalURL,
		SingingKey:      private,
		VerificationKey: public,
		KeyID:           kid,
		JWKSURL:         cfg.ExternalURL + "/keys",
		TTL:             cfg.TokenOpts.TTL,
		Audiences:       cfg.TokenOpts.Audiences,
	}

	ctx := context.Background()

	var out storage.ReadWriter
	switch cfg.Backend {
	case "dev":
		out = &storage.Dummy{}
	case "vault":
		vc, err := vault.New(
			vault.WithAddress(cfg.VaultOpts.URL),
		)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		if cfg.VaultOpts.Token != "" {
			vc.SetToken(cfg.VaultOpts.Token)
		}
		out = &storage.Vault{
			VaultClient: vc,
			MountPath:   cfg.VaultOpts.Path,
			SecretName:  "idtoken",
			SecretKey:   "value",
		}
	}

	ctl := controller.Controller{
		Pipelines:      cfg.Pipelines,
		TokenGenerator: gen,
		Storage:        out,
		RenewBefore:    cfg.TokenOpts.RenewBefore,
	}

	err = ctl.Run(ctx)
	if err != nil {
		fmt.Println("Error starting controller", err)
		os.Exit(1)
	}

}
