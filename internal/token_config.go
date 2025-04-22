package internal

import (
	"fmt"
	"time"
)

type TokenSubjectScope string

const (
	TokenSubjectScopeNone     = ""
	TokenSubjectScopeTeam     = "team"
	TokenSubjectScopePipeline = "pipeline"
)

type TokenConfig struct {
	Team         string
	Pipeline     string
	SubjectScope TokenSubjectScope
	Audience     []string
	ExpiresIn    time.Duration
	RenewBefore  time.Duration
	Path         string
}

var DefaultTokenConfig = TokenConfig{
	Team:         "main",
	SubjectScope: TokenSubjectScopePipeline,
	Audience:     nil,
	ExpiresIn:    1 * time.Hour,
	RenewBefore:  30 * time.Minute,
	Path:         "token",
}

func (c *TokenConfig) FillWithDefaults() {
	if c.Team == "" {
		c.Team = DefaultTokenConfig.Team
	}
	if c.SubjectScope == TokenSubjectScopeNone {
		c.SubjectScope = DefaultTokenConfig.SubjectScope
	}
	if c.ExpiresIn == 0 {
		c.ExpiresIn = DefaultTokenConfig.ExpiresIn
	}
	if c.RenewBefore == 0 {
		c.RenewBefore = DefaultTokenConfig.RenewBefore
	}
	if c.Path == "" {
		c.Path = DefaultTokenConfig.Path
	}
}

func (c TokenConfig) Subject() string {
	switch c.SubjectScope {
	case TokenSubjectScopeTeam:
		return c.Team
	case TokenSubjectScopePipeline:
		return c.Team + "/" + c.Pipeline
	default:
		return ""

	}
}

func (c TokenConfig) String() string {
	return c.Team + "/" + c.Pipeline + "/" + c.Path
}

func (c TokenConfig) Validate() error {
	if c.Team == "" {
		return fmt.Errorf("team must not be empty")
	}
	if c.Pipeline == "" {
		return fmt.Errorf("pipeline must not be empty")
	}
	if c.RenewBefore >= c.ExpiresIn {
		return fmt.Errorf("renewBefore must be smaller than expiresIn")
	}
	return nil
}
