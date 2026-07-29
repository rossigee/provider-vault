PLATFORMS ?= linux_amd64
GO_REQUIRED_VERSION ?= 1.26
GO_STANDARD_VERSION ?= 1.26

BUILD_REGISTRY ?= ghcr.io/rossigee
PUBLISH_REGISTRY ?= ghcr.io/rossigee

VERSION ?= $(shell git describe --dirty --always 2>/dev/null || echo "v0.0.0")

include build/makelib/common.mk

publish.artifacts:
	@echo "Publish deferred to xpkg machinery"
