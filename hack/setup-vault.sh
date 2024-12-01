#!/bin/bash
set -e

if [ -z $VAULT_ADDR ]; then
    echo VAULT_ADDR must be set
fi

if [ -z $VAULT_TOKEN ]; then
    echo VAULT_TOKEN must be set
fi

CONCOURSE_IDP_URL=http://localhost:8080

echo Configuring JWT auth in vault

if ! vault auth list | grep -q jwt; then
    vault auth enable jwt
fi

export JWT_ACCESSOR=$(vault auth list -format json | jq -r '."jwt/".accessor')

cat pipeline-policy.hcl | envsubst | vault policy write pipeline -

vault write auth/jwt/config \
    oidc_discovery_url="$CONCOURSE_IDP_URL" \
    oidc_client_id="" \
    oidc_client_secret="" \
    default_role="pipeline"

vault write auth/jwt/role/pipeline \
    role_type="jwt" \
    bound_audiences="vault" \
    user_claim="sub" \
    claim_mappings='team=team' \
    claim_mappings='pipeline=pipeline' \
    policies=pipeline \
    ttl=15m
