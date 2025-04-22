package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	cpidp "github.com/dbaumgarten/concourse-pipeline-idp/internal"
	"github.com/hashicorp/vault-client-go"
)

func main() {

	cfg, err := cpidp.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	err = cfg.Validate()
	if err != nil {
		log.Fatal("Config is invalid: ", err)
	}

	ctx := context.Background()

	var out cpidp.Storage
	switch cfg.Backend {
	case "dev":
		out = &cpidp.Dummy{}
	case "vault":
		out = getVaultStorage(cfg)
	}

	if cfg.ListenAddr != "" {
		server := cpidp.NewJWKSServer(out, cfg.ExternalURL)
		go server.ListenAndServe(cfg.ListenAddr)
	}

	if cfg.LeaderElectionOpts.Enabled {
		log.Println("Trying to aquire leader lock")
		err = cpidp.AquireLockAndHold(ctx, out, cfg.LeaderElectionOpts.Name, cfg.LeaderElectionOpts.TTL, time.Duration(float64(cfg.LeaderElectionOpts.TTL)*0.1))
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

	tokenGenerator := cpidp.NewTokenGenerator(cfg.ExternalURL, nil)

	keyManager := cpidp.KeyManager{
		Storage:           out,
		TokenGenerator:    tokenGenerator,
		KeyRotationPeriod: cfg.KeyOpts.RotationPeriod,
		KeyMaxAge:         cfg.KeyOpts.MaxAge,
	}

	// Run the keyManager once to make sure signing-keys exist and tokenGenerator is configured with a key
	_, err = keyManager.ManageOnce(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// run the keyManager in background to periodically generate new keys
	go keyManager.Manage(ctx)

	ctl := cpidp.Controller{
		TokenGenerator: tokenGenerator,
		Storage:        out,
		TokenConfigs:   cfg.Tokens,
	}

	err = ctl.Run(ctx)
	if err != nil {
		fmt.Println("Error starting controller", err)
		os.Exit(1)
	}
}

func getVaultStorage(cfg cpidp.Config) cpidp.Storage {
	vc, err := vault.New(
		vault.WithAddress(cfg.VaultOpts.URL),
	)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.VaultOpts.Token != "" {
		vc.SetToken(cfg.VaultOpts.Token)
	}
	return &cpidp.Vault{
		VaultClient:   vc,
		ConcoursePath: cfg.VaultOpts.ConcoursePath,
		ConfigPath:    cfg.VaultOpts.ConfigPath,
	}
}
