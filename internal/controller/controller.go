package controller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/token"
)

type Controller struct {
	Pipelines      []concourse.Pipeline
	TokenGenerator *token.Generator
	Storage        storage.ReadWriter
	RenewBefore    time.Duration

	cache map[string]cacheEntry
}

type cacheEntry struct {
	Token   string
	RenewAt time.Time
}

func (c *Controller) Run(ctx context.Context) error {
	for {
		if err := c.RunOnce(ctx); err != nil {
			return err
		}
		nextRun := c.getNextRenewalTime()
		delay := time.Until(nextRun)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

func (c *Controller) RunOnce(ctx context.Context) error {
	if c.cache == nil {
		if err := c.populateCache(ctx); err != nil {
			return err
		}
	}

	for _, p := range c.Pipelines {
		renewed, err := c.handlePipeline(ctx, p)
		if err != nil {
			log.Printf("Error when renewing token for pipeline %s", p)
		} else if renewed {
			log.Printf("Renewed token for pipeline %s", p)
		}
	}
	return nil
}

func (c *Controller) populateCache(ctx context.Context) error {
	c.cache = make(map[string]cacheEntry)

	for _, p := range c.Pipelines {
		currentToken, err := c.Storage.ReadToken(ctx, p)
		if err == nil {
			isValid, validUntil, err := c.TokenGenerator.IsTokenStillValid(currentToken)
			if err == nil && isValid {
				log.Printf("Found existing valid token for pipeline %s", p)
				c.cache[p.String()] = cacheEntry{
					Token:   currentToken,
					RenewAt: c.calculateRenewalTime(validUntil),
				}
			}
		} else if err != storage.ErrTokenNotFound {
			return err
		}
	}

	return nil
}

func (c *Controller) handlePipeline(ctx context.Context, p concourse.Pipeline) (bool, error) {
	if c.pipelineNeedsNewToken(p) {
		newToken, validUntil, err := c.TokenGenerator.Generate(p)
		if err != nil {
			return false, fmt.Errorf("error when generating new token for pipeline %s: %w", p, err)
		}

		err = c.Storage.WriteToken(ctx, p, newToken)
		if err != nil {
			return false, fmt.Errorf("error when storing new token for pipeline %s: %w", p, err)
		}

		c.cache[p.String()] = cacheEntry{
			Token:   newToken,
			RenewAt: c.calculateRenewalTime(validUntil),
		}

		return true, nil
	}
	return false, nil
}

func (c Controller) pipelineNeedsNewToken(p concourse.Pipeline) bool {
	if cached, exists := c.cache[p.String()]; exists {
		if time.Now().Before(cached.RenewAt) {
			return false
		}
	}
	return true
}

func (c Controller) calculateRenewalTime(validUntil time.Time) time.Time {
	return validUntil.Add(-(c.RenewBefore - 2*time.Second))
}

func (c Controller) getNextRenewalTime() time.Time {
	next := time.Now().Add(24 * time.Hour)
	for _, entry := range c.cache {
		if entry.RenewAt.Before(next) {
			next = entry.RenewAt
		}
	}
	return next
}
