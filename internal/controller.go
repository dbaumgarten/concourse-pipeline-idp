package internal

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Controller struct {
	TokenConfigs   []TokenConfig
	TokenGenerator *TokenGenerator
	Storage        Storage

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

	for _, t := range c.TokenConfigs {
		renewed, err := c.handleTokenConfig(ctx, t)
		if err != nil {
			log.Printf("Error when renewing token %s", t)
		} else if renewed {
			log.Printf("Renewed token %s", t)
		}
	}
	return nil
}

func (c *Controller) populateCache(ctx context.Context) error {
	c.cache = make(map[string]cacheEntry)

	for _, t := range c.TokenConfigs {
		currentToken, err := c.Storage.ReadToken(ctx, t)
		if err == nil {
			isValid, validUntil, err := c.TokenGenerator.IsTokenStillValid(currentToken)
			if err == nil && isValid {
				log.Printf("Found existing valid token %s", t)
				c.cache[t.String()] = cacheEntry{
					Token:   currentToken,
					RenewAt: c.calculateRenewalTime(validUntil, t.RenewBefore),
				}
			}
		} else if err != ErrTokenNotFound {
			return err
		}
	}

	return nil
}

func (c *Controller) handleTokenConfig(ctx context.Context, t TokenConfig) (bool, error) {
	if c.tokenNeedsToBeRenewed(t) {
		newToken, validUntil, err := c.TokenGenerator.Generate(t)
		if err != nil {
			return false, fmt.Errorf("error when generating new token %s: %w", t, err)
		}

		err = c.Storage.WriteToken(ctx, t, newToken)
		if err != nil {
			return false, fmt.Errorf("error when storing new token %s: %w", t, err)
		}

		c.cache[t.String()] = cacheEntry{
			Token:   newToken,
			RenewAt: c.calculateRenewalTime(validUntil, t.RenewBefore),
		}

		return true, nil
	}
	return false, nil
}

func (c Controller) tokenNeedsToBeRenewed(t TokenConfig) bool {
	if cached, exists := c.cache[t.String()]; exists {
		if time.Now().Before(cached.RenewAt) {
			return false
		}
	}
	return true
}

func (c Controller) calculateRenewalTime(validUntil time.Time, renewBefore time.Duration) time.Time {
	return validUntil.Add(-(renewBefore - 2*time.Second))
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
