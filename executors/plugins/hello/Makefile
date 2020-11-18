
##### ^^^^^^   USAGE ^^^^^^ #####

##### ^^^^^^ make help ^^^^^^ #####

TARGET_NAME 			= hello
BUILD_MODE				= -buildmode=plugin

# If you want to enable the cross-compilation, uncomment the following line:
ENABLE_CROSS_COMPILATION := true

##### ^^^^^^ EDIT ABOVE ^^^^^^ #####

##### =====> Commands <===== #####

include ../../../.build/core.mk
include ../../../.build/go.mk

.PHONY: clean

build: $(CROSS_COMPILED_BINARIES) ## Run build

clean:
	@rm -rf dist