PROJECT_NAME := provider-vault
PROJECT_REPO := github.com/rossigee/$(PROJECT_NAME)
CROSSPLANE_VERSION = 2.5.0

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

-include build/makelib/output.mk

GO_REQUIRED_VERSION ?= 1.27.1
GOLANGCILINT_VERSION ?= 2.13.2
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
xpkg.release.publish.ghcr.io/rossigee.provider-vault:
	@$(foreach p,$(XPKG_LINUX_PLATFORMS),$(MAKE) xpkg.build.provider-vault PLATFORM=$(p) || exit 1;)
	@$(CROSSPLANE_CLI) xpkg push \
		$(foreach p,$(XPKG_LINUX_PLATFORMS),--package-files $(XPKG_OUTPUT_DIR)/$(p)/provider-vault-$(VERSION).xpkg ) \
		ghcr.io/rossigee/provider-vault:$(VERSION)
	@$(OK) Pushed package ghcr.io/rossigee/provider-vault:$(VERSION)


-include build/makelib/local.xpkg.mk

# Neutralize the plain runtime image push. imagelight.mk injects
# img.release.publish.<reg>.<img> as a publish.artifacts prerequisite on
# release branches, and cluster/images img.publish would fail because the
# runtime image is never built/tagged locally. The runtime binary is already
# embedded in the xpkg, so publishing this image would only overwrite the
# xpkg's package.yaml.
img.release.publish.ghcr.io/rossigee.provider-vault:
	@:
