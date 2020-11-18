
##### ^^^^^^   USAGE ^^^^^^ #####

##### ^^^^^^ make help ^^^^^^ #####

TARGET_NAME 			= venom
LDFLAGS					= -ldflags "-X github.com/ovh/venom.Version=$(VERSION)"

# If you want to enable the cross-compilation, uncomment the following line:
ENABLE_CROSS_COMPILATION := true

##### ^^^^^^ EDIT ABOVE ^^^^^^ #####

##### =====> Commands <===== #####

include ../../.build/core.mk
include ../../.build/go.mk

.PHONY: clean

build: $(CROSS_COMPILED_BINARIES) ## Run build

clean:
	@rm -f venom
	@rm -rf dist