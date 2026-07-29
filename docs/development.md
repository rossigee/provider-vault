# Development

## Prerequisites

- Go 1.26+
- Docker
- Make

## Build

```bash
# Build the binary
make build

# Build and publish with version
make publish VERSION=v0.2.7 PLATFORMS=linux_amd64
```

## Test

```bash
make test
make lint
make reviewable
```

## Generate CRDs

```bash
make generate
```

## Project Structure

```
provider-vault/
├── apis/               # CRD type definitions (one per resource)
│   └── <resource>/v1beta1/
├── cmd/provider/       # Entry point (main.go)
├── config/             # Provider configuration and setup
├── internal/
│   ├── clients/        # Vault HTTP client (raw net/http, no SDK)
│   ├── controller/     # Reconciler logic (one per resource)
│   └── recorder/       # Event recorder
├── package/            # Crossplane packaging
│   ├── crossplane.yaml
│   └── crds/           # Generated CRD YAML
├── cluster/images/     # Dockerfile
└── examples/           # Usage examples
```

## Architecture

- **Raw HTTP Client**: Uses `net/http` directly instead of the official Vault Go SDK to avoid transitive dependency conflicts with Crossplane.
- **Crossplane v2 Namespaced Resources**: All managed resources are namespace-scoped for multi-tenancy.
- **Per-Resource Controllers**: Each resource type has an independent reconciler registered at startup.

## Adding a New Resource

1. Create `apis/<resource>/v1beta1/<resource>_types.go` with Spec, Status, and Parameters structs
2. Register the type in `apis/<resource>/v1beta1/register.go`
3. Run `make generate` to produce CRDs and generated client code
4. Create `internal/controller/<resource>/<resource>.go` with connector + CRUD logic
5. Register the controller in `internal/controller/<resource>/<resource>.go:Setup()`
6. Wire it up in `internal/controller/setup.go`
7. Add examples in `examples/<resource>/`
8. Add documentation in `docs/resources/`

## Vault API Client

The provider uses a raw HTTP client (`internal/clients/vault.go`) to communicate with Vault. Each API call:

1. Builds the URL path (e.g. `/v1/secret/data/my-path`)
2. Constructs the request body as `map[string]interface{}`
3. Sends the request with the Vault token in the `X-Vault-Token` header
4. Parses the JSON response

## Releasing

```bash
# Build and publish
make publish VERSION=v0.2.7 PLATFORMS=linux_amd64

# Update the provider in your cluster
kubectl patch provider provider-vault --type='merge' \
  -p='{"spec":{"package":"ghcr.io/rossigee/provider-vault:v0.2.7"}}'
```

## Troubleshooting Build Issues

### Docker cache using stale binary

After `make clean`, the Docker build cache may still contain old layers. Clear it:

```bash
docker buildx prune -af
```

### "No command specified" in provider pod

Check `package/crossplane.yaml` exists (not `package.yaml`). The build system requires `crossplane.yaml` for Docker image embedding.
