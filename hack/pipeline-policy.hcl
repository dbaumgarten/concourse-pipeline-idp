# allow a pipeline that is authenticated via JWT, to list/read all of it's secrets


path "secret/metadata/concourse/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.team }}" {
  capabilities = ["list"]
}

path "secret/data/concourse/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.team }}/+" {
  capabilities = ["read"]
}

path "secret/metadata/concourse/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.team }}/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.pipeline }}" {
  capabilities = ["list"]
}

path "secret/metadata/concourse/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.team }}/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.pipeline }}/*" {
  capabilities = ["read", "list"]
}

path "secret/data/concourse/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.team }}/{{ identity.entity.aliases.$JWT_ACCESSOR.metadata.pipeline }}/*" {
  capabilities = ["read", "list"]
}
