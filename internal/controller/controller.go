package controller

import (
	"context"
	"log"
	"time"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/pipeline"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/storage"
	"github.com/dbaumgarten/concourse-pipeline-idp/internal/token"
)

type Controller struct {
	Pipeline       pipeline.ConcoursePipeline
	TokenGenerator token.Generator
	Storage        storage.ReadWriter
	RenewBefore    time.Duration
}

func (m Controller) Run(ctx context.Context) {

	var delay time.Duration

	currentToken, err := m.Storage.ReadToken(ctx, m.Pipeline)
	if err != nil {
		if err == storage.ErrTokenNotFound {
			log.Printf("No existing token found for pipeline %s", m.Pipeline)
		}
	} else {
		isValid, remainingValidity, err := m.TokenGenerator.IsTokenStillValid(currentToken)
		if err != nil {
			log.Printf("Error verifying existing token for pipeline %s: %s", m.Pipeline, err)
		}
		if isValid {
			delay = m.calucalteDelay(remainingValidity)
		}
	}

	for {
		if delay > 0 {
			log.Printf("Sleeping %d seconds before handling %s again", int(delay.Seconds()), m.Pipeline)
			select {
			case <-ctx.Done():
				return
			case <-time.NewTimer(delay).C:
				// delay passed, handle pipeline
			}
		}

		log.Printf("Generating new token for pipeline %s", m.Pipeline)

		newToken, validity, err := m.TokenGenerator.Generate(m.Pipeline)
		if err != nil {
			log.Printf("Error generating new token for pipeline %s: %s", m.Pipeline, err)
		}
		currentToken = newToken
		delay = m.calucalteDelay(validity)

		err = m.Storage.WriteToken(ctx, m.Pipeline, currentToken)
		if err != nil {
			log.Printf("Error storing new token for pipeline %s: %s", m.Pipeline, err)
		}

		log.Printf("Refreshed token for pipeline %s", m.Pipeline)
	}
}

func (m Controller) calucalteDelay(remainingValidity time.Duration) time.Duration {
	return remainingValidity - m.RenewBefore - 1*time.Second
}
