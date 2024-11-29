package storage

import (
	"context"
	"encoding/json"
	"path"
	"strings"

	"github.com/dbaumgarten/concourse-pipeline-idp/internal/concourse"
	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type Vault struct {
	VaultClient   *vault.Client
	ConcoursePath string
	ConfigPath    string
}

func (v Vault) WriteToken(ctx context.Context, p concourse.Pipeline, token string) error {
	mountpoint, basepath := splitPath(v.ConcoursePath)
	targetPath := path.Join(basepath, p.Team, p.Name, "idtoken")

	_, err := v.VaultClient.Secrets.KvV2Write(ctx, targetPath, schema.KvV2WriteRequest{
		Data: map[string]interface{}{
			"value": token,
		}},
		vault.WithMountPath(mountpoint),
	)
	return err
}

func (v Vault) ReadToken(ctx context.Context, p concourse.Pipeline) (string, error) {
	mountpoint, basepath := splitPath(v.ConcoursePath)
	targetPath := path.Join(basepath, p.Team, p.Name, "idtoken")

	secret, err := v.VaultClient.Secrets.KvV2Read(ctx, targetPath, vault.WithMountPath(mountpoint))
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return "", ErrTokenNotFound
		}
		return "", err
	}

	return secret.Data.Data["value"].(string), nil
}

func (v Vault) StoreKey(ctx context.Context, key jwk.Key) error {
	encoded, err := json.Marshal(key)
	if err != nil {
		return err
	}

	kid, _ := key.KeyID()
	mountpoint, basepath := splitPath(v.ConfigPath)
	targetPath := path.Join(basepath, "keys")

	_, err = v.VaultClient.Secrets.KvV2Write(ctx, targetPath, schema.KvV2WriteRequest{
		Data: map[string]interface{}{
			kid: string(encoded),
		}},
		vault.WithMountPath(mountpoint),
	)

	return err
}

func (v Vault) GetKeys(ctx context.Context) (jwk.Set, error) {
	mountpoint, basepath := splitPath(v.ConfigPath)
	targetPath := path.Join(basepath, "keys")

	keys, err := v.VaultClient.Secrets.KvV2Read(ctx, targetPath, vault.WithMountPath(mountpoint))
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return jwk.NewSet(), nil
		}
		return nil, err
	}
	set := jwk.NewSet()
	for _, key := range keys.Data.Data {
		parsed, err := jwk.ParseKey([]byte(key.(string)))
		if err != nil {
			return nil, err
		}
		err = set.AddKey(parsed)
		if err != nil {
			return nil, err
		}
	}
	return set, nil
}

func splitPath(spath string) (string, string) {
	parts := strings.SplitN(spath, "/", 2)
	switch len(parts) {
	case 2:
		return parts[0], parts[1]
	case 1:
		return parts[0], ""
	}
	return "", ""
}
