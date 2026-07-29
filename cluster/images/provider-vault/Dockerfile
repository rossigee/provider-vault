FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.title="provider-vault"
LABEL org.opencontainers.image.description="Crossplane provider for HashiCorp Vault secrets management"
LABEL org.opencontainers.image.vendor="rossigee"

COPY --chmod=0755 bin/linux_amd64/provider /usr/local/bin/provider
COPY --chmod=0644 package.yaml /package.yaml
COPY --chmod=0644 crds /crds

ENTRYPOINT ["/usr/local/bin/provider"]
