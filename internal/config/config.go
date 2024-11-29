package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	ExternalURL string
	ListenAddr  string
	Backend     string
	ConcourseOpts
	TokenOpts
	VaultOpts
}

type ConcourseOpts struct {
	Pipelines []concourse.Pipeline
}

type VaultOpts struct {
	URL           string
	Token         string
	ApproleID     string
	ApproleSecret string
	Path          string
}

type TokenOpts struct {
	TTL         time.Duration
	RenewBefore time.Duration
	Audiences   []string
}

func LoadConfig() (Config, error) {

	flag.String("externalURL", "", "Under which URL the server will be reachable for external services")
	flag.String("listenAddr", ":8080", "Where to listen on for the JWKS Server")
	flag.String("backend", "vault", "Which storage-backend to use [vault,dev]")

	flag.StringSlice("concourse.pipelines", []string{}, "List of pipelines in format <team>/<pipeline> for which to manage tokens")

	flag.Duration("token.ttl", 1*time.Hour, "How long issued tokens are valid")
	flag.Duration("token.renewBefore", 30*time.Minute, "How long before their expiry tokens should be renewed")
	flag.StringSlice("token.audiences", []string{"concourse-pipeline-idp"}, "Which audiences to include in the tokens")

	flag.String("vault.url", "", "URL under which vault is reachable")
	flag.String("vault.token", "", "Token used to authenticate with vault")
	flag.String("vault.approleid", "", "RoleID for approle authentication")
	flag.String("vault.approlesecret", "", "Secret for approle authentication")
	flag.String("vault.path", "/concourse", "Path uder which the concourse-secrets can be found in vault")

	flag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/concourse-pipeline-idp")
	viper.AddConfigPath("$HOME/.concourse-pipeline-idp")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("CPIDP_")
	viper.AutomaticEnv()
	viper.BindPFlags(flag.CommandLine)

	err := viper.ReadInConfig()
	if err != nil {
		// silentily ignore error
	}

	cfg := Config{
		ExternalURL: viper.GetString("externalURL"),
		ListenAddr:  viper.GetString("listenAddr"),
		Backend:     viper.GetString("backend"),
		ConcourseOpts: ConcourseOpts{
			Pipelines: make([]concourse.Pipeline, 0, 10),
		},
		TokenOpts: TokenOpts{
			TTL:         viper.GetDuration("token.ttl"),
			RenewBefore: viper.GetDuration("token.renewBefore"),
			Audiences:   viper.GetStringSlice("token.audiences"),
		},
		VaultOpts: VaultOpts{
			URL:           viper.GetString("vault.url"),
			Token:         viper.GetString("vault.token"),
			ApproleID:     viper.GetString("vault.approleid"),
			ApproleSecret: viper.GetString("vault.approlesecret"),
			Path:          viper.GetString("vault.path"),
		},
	}

	for _, p := range viper.GetStringSlice("concourse.pipelines") {
		parts := strings.Split(p, "/")
		if len(parts) == 2 {
			cfg.Pipelines = append(cfg.Pipelines, concourse.Pipeline{
				Team: parts[0],
				Name: parts[1],
			})
		}
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.ExternalURL == "" {
		return fmt.Errorf("externalURL must be set")
	}
	if c.TokenOpts.RenewBefore >= c.TokenOpts.TTL {
		return fmt.Errorf("token.renewBefore must be smaller than token.ttl")
	}
	if len(c.Pipelines) == 0 {
		return fmt.Errorf("concourse.pipelines must have at least on element")
	}
	if c.Backend != "dev" && c.Backend != "vault" {
		return fmt.Errorf("backend must either be dev or vault")
	}
	if c.Backend == "vault" {
		if c.VaultOpts.URL == "" {
			return fmt.Errorf("vault.url must be set")
		}
		if c.VaultOpts.Token == "" && (c.VaultOpts.ApproleID == "" || c.VaultOpts.ApproleSecret == "") {
			return fmt.Errorf("vault.token or vault.approleid+vault.approlesecret must be set")
		}
	}
	return nil
}
