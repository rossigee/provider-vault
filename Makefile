PROJECT_NAME := provider-vault
PROJECT_REPO := github.com/rossigee/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64
-include build/makelib/common.mk

-include build/makelib/output.mk

GO_REQUIRED_VERSION ?= 1.26.6
GOLANGCILINT_VERSION ?= 2.12.2
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
-include build/makelib/golang.mk

-include build/makelib/k8s_tools.mk

IMAGES = provider-vault
REGISTRY_ORGS = ghcr.io/rossigee
-include build/makelib/imagelight.mk

XPKG_REG_ORGS = ghcr.io/rossigee
XPKGS = provider-vault
-include build/makelib/xpkg.mk

xpkg.build.provider-vault: do.build.images

$(foreach x,$(XPKGS),$(eval xpkg.build.$(x): $(CROSSPLANE_CLI)))

$(foreach p,$(filter linux_%,$(PLATFORMS)),$(foreach x,$(XPKGS),$(eval $(XPKG_OUTPUT_DIR)/$(p)/$(x)-$(VERSION).xpkg: $(CROSSPLANE_CLI); @$(MAKE) xpkg.build.$(x) PLATFORM=$(p))))

$(foreach r,$(XPKG_REG_ORGS),$(foreach x,$(XPKGS),$(eval xpkg.release.publish.$(r).$(x): $(CROSSPLANE_CLI) $(foreach p,$(filter linux_%,$(PLATFORMS)),$(XPKG_OUTPUT_DIR)/$(p)/$(x)-$(VERSION).xpkg))))

publish.artifacts:
	$(foreach r,$(XPKG_REG_ORGS), $(foreach x,$(XPKGS),@$(MAKE) xpkg.release.publish.$(r).$(x)))

-include build/makelib/local.xpkg.mk
