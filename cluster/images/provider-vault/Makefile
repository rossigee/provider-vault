PLATFORMS := linux_amd64 linux_arm64
include ../../../build/makelib/common.mk

IMAGE ?= $(BUILD_REGISTRY)/provider-vault:$(VERSION)

include ../../../build/makelib/imagelight.mk

img.build: ../../../_output/bin/$(PLATFORM)/provider
	@$(MAKE) BUILD_ARGS="--load --no-cache" img.build.shared

../../../_output/bin/$(PLATFORM)/provider:
	@$(MAKE) -C ../../.. go.build PLATFORM=$(PLATFORM)

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/
	@cp ../../../package/crossplane.yaml $(IMAGE_TEMP_DIR)/package.yaml
	@cp -r ../../../package/crds $(IMAGE_TEMP_DIR)/crds
	@docker buildx build $(BUILD_ARGS) --platform $(IMAGE_PLATFORM) -t $(IMAGE) $(IMAGE_TEMP_DIR)

img.publish:
	@echo Publish is deferred to xpkg machinery
