package internal

import (
	"fmt"
	"os"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	ExternalURL        string
	ListenAddr         string
	Backend            string
	VaultOpts          VaultOpts
	LeaderElectionOpts LeaderElectionOpts
	KeyOpts            KeyOpts
	Tokens             []TokenConfig
}

type VaultOpts struct {
	URL           string
	Token         string
	ApproleID     string
	ApproleSecret string
	ConcoursePath string
	ConfigPath    string
}

type KeyOpts struct {
	RotationPeriod time.Duration
	MaxAge         time.Duration
}

type LeaderElectionOpts struct {
	Enabled bool
	Name    string
	TTL     time.Duration
}

func LoadConfig() (Config, error) {
	flag.String("externalUrl", "", "Under which URL the server will be reachable for external services")
	flag.String("listenAddr", ":8080", "Where to listen on for the JWKS Server")
	flag.String("backend", "vault", "Which storage-backend to use [vault,dev]")

	flag.StringSlice("concourse.pipelines", []string{}, "List of pipelines in format <team>/<pipeline> for which to manage tokens")

	flag.String("vault.url", "", "URL under which vault is reachable")
	flag.String("vault.token", "", "Token used to authenticate with vault")
	flag.String("vault.approleId", "", "RoleID for approle authentication")
	flag.String("vault.approleSecret", "", "Secret for approle authentication")
	flag.String("vault.concoursePath", "/concourse", "Path under which the concourse-secrets can be found in vault")
	flag.String("vault.configPath", "/concourse/pipeline-idp", "Path under which the store config for this tool in vault")

	flag.Bool("leaderElection.enabled", true, "Whether to use the storage-backend to elect a leader (required for HA-Setup)")
	flag.String("leaderElection.name", "", "Name to use for this instance during leader-election. Must be unique, defaults to hostname")
	flag.Duration("leaderElection.ttl", 1*time.Minute, "How long a leaderElection remains valid")

	flag.Duration("key.rotationPeriod", 24*time.Hour, "Time after which a new signing key should be generated and used")
	flag.Duration("key.maxAge", 48*time.Hour, "Time after which a key should be removed from the jwks")

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
		ExternalURL: viper.GetString("externalUrl"),
		ListenAddr:  viper.GetString("listenAddr"),
		Backend:     viper.GetString("backend"),
		VaultOpts: VaultOpts{
			URL:           viper.GetString("vault.url"),
			Token:         viper.GetString("vault.token"),
			ApproleID:     viper.GetString("vault.approleId"),
			ApproleSecret: viper.GetString("vault.approleSecret"),
			ConcoursePath: viper.GetString("vault.concoursePath"),
			ConfigPath:    viper.GetString("vault.configPath"),
		},
		LeaderElectionOpts: LeaderElectionOpts{
			Enabled: viper.GetBool("leaderElection.enabled"),
			Name:    viper.GetString("leaderElection.name"),
			TTL:     viper.GetDuration("leaderElection.ttl"),
		},
		KeyOpts: KeyOpts{
			RotationPeriod: viper.GetDuration("key.rotationPeriod"),
			MaxAge:         viper.GetDuration("key.maxAge"),
		},
		Tokens: []TokenConfig{},
	}
	err = viper.UnmarshalKey("tokens", &cfg.Tokens)
	if err != nil {
		return Config{}, err
	}

	for i := range cfg.Tokens {
		cfg.Tokens[i].FillWithDefaults()
	}

	if cfg.LeaderElectionOpts.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		cfg.LeaderElectionOpts.Name = hostname
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.ExternalURL == "" {
		return fmt.Errorf("externalURL must be set")
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
	for _, tokenConfig := range c.Tokens {
		if err := tokenConfig.Validate(); err != nil {
			return fmt.Errorf("invalid token config: %w", err)
		}
	}
	if c.KeyOpts.MaxAge <= c.KeyOpts.RotationPeriod {
		return fmt.Errorf("key.maxAge must be larger than key.rotationPeriod")
	}
	return nil
}
