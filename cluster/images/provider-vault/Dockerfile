FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.title="provider-vault"
LABEL org.opencontainers.image.description="Crossplane provider for HashiCorp Vault secrets management"
LABEL org.opencontainers.image.vendor="rossigee"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/rossigee/provider-vault"
LABEL org.opencontainers.image.url="https://github.com/rossigee/provider-vault"
LABEL org.opencontainers.image.documentation="https://github.com/rossigee/provider-vault/blob/master/README.md"

COPY --chmod=0755 bin/linux_amd64/provider /usr/local/bin/provider
COPY --chmod=0644 package.yaml /package.yaml
COPY --chmod=0644 crds /crds

ENTRYPOINT ["/usr/local/bin/provider"]
