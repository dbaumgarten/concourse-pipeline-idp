package storage

import (
	"context"
	"path"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/pipeline"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type Vault struct {
	VaultClient       *vault.Client
	MountPath         string
	ConcourseBasepath string
	SecretName        string
	SecretKey         string
}

func (v Vault) WriteToken(ctx context.Context, p pipeline.ConcoursePipeline, token string) error {
	targetPath := path.Join(p.Team, p.Name, v.SecretName)

	_, err := v.VaultClient.Secrets.KvV2Write(ctx, targetPath, schema.KvV2WriteRequest{
		Data: map[string]interface{}{
			v.SecretKey: token,
		}},
		vault.WithMountPath(v.MountPath),
	)
	return err
}

func (v Vault) ReadToken(ctx context.Context, p pipeline.ConcoursePipeline) (string, error) {
	targetPath := path.Join(p.Team, p.Name, v.SecretName)

	secret, err := v.VaultClient.Secrets.KvV2Read(ctx, targetPath, vault.WithMountPath(v.MountPath))
	if err != nil {
		return "", err
	}

	return secret.Data.Data[v.SecretKey].(string), nil
}
