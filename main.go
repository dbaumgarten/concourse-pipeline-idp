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
	"github.com/hashicorp/vault-client-go"
)

func main() {

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	err = cfg.Validate()
	if err != nil {
		log.Fatal("Config is invalid: ", err)
	}

	ctx := context.Background()

	var out storage.Storage
	switch cfg.Backend {
	case "dev":
		out = &storage.Dummy{}
	case "vault":
		out = getVaultStorage(cfg)
	}

	signingKeys, existing, err := keys.LoadOrGenerateAndStoreKeys(ctx, out)
	if err != nil {
		log.Fatal(err)
	}

	if existing {
		log.Println("Found existing signing key(s)")
	} else {
		log.Println("No existing keys found! Generating new one")
	}

	if cfg.ListenAddr != "" {
		server := keys.NewJWKSServer(out)
		go server.ListenAndServe(cfg.ListenAddr)
	}

	newestKey := keys.FindNewestKey(signingKeys)
	kid, _ := newestKey.KeyID()
	log.Println("Using key with kid:", kid)

	gen := &token.Generator{
		Issuer:    cfg.ExternalURL,
		Key:       newestKey,
		TTL:       cfg.TokenOpts.TTL,
		Audiences: cfg.TokenOpts.Audiences,
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

func getVaultStorage(cfg config.Config) storage.Storage {
	vc, err := vault.New(
		vault.WithAddress(cfg.VaultOpts.URL),
	)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.VaultOpts.Token != "" {
		vc.SetToken(cfg.VaultOpts.Token)
	}
	return &storage.Vault{
		VaultClient:   vc,
		ConcoursePath: cfg.VaultOpts.ConcoursePath,
		ConfigPath:    cfg.VaultOpts.ConfigPath,
	}
}
