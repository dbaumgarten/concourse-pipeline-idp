package manager

import (
	"context"
	"log"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/pipeline"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/token"
)

type Manager struct {
	Interval       time.Duration
	RenewBefore    time.Duration
	Pipelines      pipeline.Lister
	TokenGenerator token.Generator
	Storage        storage.ReadWriter
}

func (m Manager) ManageLoop(ctx context.Context) error {
	for {

		m.Manage(ctx)
		log.Printf("Processed all pipelines. Sleeping for %d seconds", int(m.Interval.Seconds()))
		select {
		case <-ctx.Done():
			return nil
		case <-time.NewTimer(m.Interval).C:
			continue
		}
	}
}

func (m Manager) Manage(ctx context.Context) error {
	pipelines, err := m.Pipelines.List(ctx)
	if err != nil {
		return err
	}

	for _, p := range pipelines {
		log.Printf("Processing pipeline %s\n", p.String())
		renewed, err := m.handlePipeline(ctx, p)
		if err != nil {
			log.Printf("Failed to renew token for pipeline %s: %s\n", p.String(), err)
		}
		if renewed {
			log.Printf("Renewed token for pipeline %s\n", p.String())
		} else {
			log.Printf("Token for pipeline %s is new enough. Doing nothing!\n", p.String())
		}
	}

	return nil
}

func (m Manager) handlePipeline(ctx context.Context, p pipeline.ConcoursePipeline) (bool, error) {
	if m.pipelineNeedsNewToken(ctx, p) {
		newToken, err := m.TokenGenerator.Generate(p)
		if err != nil {
			return false, err
		}
		err = m.Storage.WriteToken(ctx, p, newToken)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (m Manager) pipelineNeedsNewToken(ctx context.Context, p pipeline.ConcoursePipeline) bool {
	currentToken, err := m.Storage.ReadToken(ctx, p)
	if err != nil {
		if err == storage.ErrTokenNotFound {
			log.Printf("No token found for pipeline %s", p.String())
			return true
		}
	}

	isValid, remainingValidity, err := m.TokenGenerator.IsTokenStillValid(currentToken)
	if err != nil {
		log.Printf("Error verifying existing token: %s", err)
		return true
	}

	if !isValid {
		log.Printf("Existing token for pipeline %s is not valid (anymore)", p.String())
		return true
	}

	if remainingValidity < m.RenewBefore {
		log.Printf("Existing token for pipeline %s will expire soon", p.String())
		return true
	}

	return false
}
