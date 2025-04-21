package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	if cfg.LeaderElectionOpts.Enabled {
		log.Println("Trying to aquire leader lock")
		err = storage.LockAndHold(ctx, out, cfg.LeaderElectionOpts.Name, cfg.LeaderElectionOpts.TTL, time.Duration(float64(cfg.LeaderElectionOpts.TTL)*0.1))
		if err != nil {
			log.Fatal("Error aquiring leader lock", err)
		}
		log.Println("Aquired leader lock")

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			log.Print("Releasing leader lock")
			out.ReleaseLock(ctx)
			os.Exit(0)
		}()
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
		server := keys.NewJWKSServer(out, cfg.ExternalURL)
		go server.ListenAndServe(cfg.ListenAddr)
	}

	newestKey := keys.FindNewestKey(signingKeys)
	log.Println("Using key with kid:", newestKey.KeyID)

	gen := &token.Generator{
		Issuer:    cfg.ExternalURL,
		Key:       *newestKey,
		TTL:       cfg.TokenOpts.TTL,
		Audiences: cfg.TokenOpts.Audiences,
	}

	ctl := controller.Controller{
		Pipelines:      cfg.ConcourseOpts.Pipelines,
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
