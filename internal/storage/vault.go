package storage

import (
	"context"
	"encoding/json"
	"log"
	"path"
	"strconv"
	"strings"
	"time"

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

type lock struct {
	Name    string
	Until   time.Time
	Version int64
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
			return jwk.NewSet(), ErrNoKeysFound
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

func (v Vault) Lock(ctx context.Context, name string, duration time.Duration) error {
	for {
		curentLock, err := v.getCurrentLock(ctx)
		if err != nil {
			return err
		}

		var currentVersion int64

		if curentLock != nil {
			if curentLock.Until.After(time.Now()) && curentLock.Name != name {
				// sleep until the existing lock expires
				duration := time.Until(curentLock.Until)
				log.Printf("Lock is already held by %s, sleeping for %s until retry", curentLock.Name, duration.String())
				time.Sleep(duration)
			}
			currentVersion = curentLock.Version
		}

		newlock := lock{
			Name:    name,
			Until:   time.Now().Add(duration),
			Version: currentVersion,
		}
		err = v.tryAquireLock(ctx, newlock)
		if err == nil {
			return nil
		}
	}
}

func (v Vault) getCurrentLock(ctx context.Context) (*lock, error) {
	mountpoint, basepath := splitPath(v.ConfigPath)
	targetPath := path.Join(basepath, "lock")

	resp, err := v.VaultClient.Secrets.KvV2Read(ctx, targetPath, vault.WithMountPath(mountpoint))
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return nil, nil
		} else {
			return nil, err
		}
	}

	i, _ := strconv.ParseInt(resp.Data.Data["exp"].(string), 10, 64)
	curVer := resp.Data.Metadata["version"]
	curVerInt, _ := curVer.(json.Number).Int64()

	return &lock{
		Name:    resp.Data.Data["sub"].(string),
		Until:   time.Unix(i, 0),
		Version: curVerInt,
	}, nil
}

func (v Vault) tryAquireLock(ctx context.Context, lock lock) error {
	mountpoint, basepath := splitPath(v.ConfigPath)
	targetPath := path.Join(basepath, "lock")

	_, err := v.VaultClient.Secrets.KvV2Write(ctx, targetPath, schema.KvV2WriteRequest{
		Options: map[string]interface{}{
			"cas": lock.Version,
		},
		Data: map[string]interface{}{
			"sub": lock.Name,
			"exp": strconv.Itoa(int(lock.Until.Unix())),
		}},
		vault.WithMountPath(mountpoint),
	)

	return err
}

func (v Vault) ReleaseLock(ctx context.Context) error {
	mountpoint, basepath := splitPath(v.ConfigPath)
	targetPath := path.Join(basepath, "lock")

	_, err := v.VaultClient.Secrets.KvV2DeleteMetadataAndAllVersions(ctx, targetPath, vault.WithMountPath(mountpoint))
	return err
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
