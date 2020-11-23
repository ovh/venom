##### ^^^^^^   USAGE ^^^^^^ #####

##### ^^^^^^ make help ^^^^^^ #####

DOCKER_IMAGE 		:= $(if ${DOCKER_IMAGE},${DOCKER_IMAGE},ovhcom/venom:snapshot)
DOCKER_IMAGE_NAME 	= $(shell echo $(DOCKER_IMAGE) | cut -d ':' -f 1)
DOCKER_IMAGE_TAG 	= $(shell echo $(DOCKER_IMAGE) | cut -d ':' -f 2)

##### ^^^^^^ EDIT ABOVE ^^^^^^ #####

include ./.build/core.mk

ifneq (,$(strip $(filter build dist run lint clean test test-xunit,$(MAKECMDGOALS))))
include ./.build/go.mk
endif

DIST_DIR := dist/
RESULTS_DIR := results/

APP_DIST := $(wildcard cmd/venom/dist/*)
ALL_DIST := $(APP_DIST)
ALL_DIST_TARGETS := $(foreach DIST,$(ALL_DIST),$(addprefix $(DIST_DIR),$(notdir $(DIST))))

APP_RESULTS = $(wildcard cmd/venom/results/*)
ALL_RESULTS = $(APP_RESULTS)
ALL_RESULTS_TARGETS := $(foreach RESULTS,$(ALL_RESULTS),$(addprefix $(RESULTS_DIR),$(notdir $(RESULTS))))

define get_dist_from_target
$(filter %/$(notdir $(1)), $(ALL_DIST))
endef

define get_results_from_target
$(filter %/$(notdir $(1)), $(ALL_RESULTS))
endef

$(ALL_DIST_TARGETS):
	@mkdir -p $(DIST_DIR)
	$(info copying $(call get_dist_from_target, $@) to $@)
	@cp -f $(call get_dist_from_target, $@) $@

$(ALL_RESULTS_TARGETS):
	@mkdir -p $(RESULTS_DIR)
	$(info copying $(call get_results_from_target, $@) to $@)
	@cp -f $(call get_results_from_target, $@) $@

.PHONY: build lint clean testrun test dist test-xunit package

build: ## build all components and push them into dist directory
	$(info Building Component venom)
	$(MAKE) build -C cmd/venom
	$(MAKE) dist

plugins: ## build all components and push them into dist directory
	$(info Building plugin)
	$(MAKE) build -C executors/plugins
	$(MAKE) dist -C executors/plugins
	@mkdir -p dist/lib && \
	mv executors/plugins/dist/lib/* dist/lib;

dist: $(ALL_DIST_TARGETS)

run: ## build binary for current OS only and run it. For development purpose only
	OS=${UNAME_LOWERCASE} $(MAKE) build -C cmd/venom
	@cmd/venom/dist/venom_${UNAME_LOWERCASE}_amd64

lint: mk_go_lint ## install and run golangci-lint on all go files. doc https://github.com/golangci/golangci-lint

clean: mk_go_clean ## delete directories dist and results and all temp files (coverage, tests, reports)
	@rm -rf ${DIST_DIR}
	@rm -rf ${RESULTS_DIR}
	$(MAKE) clean -C cmd/venom
	$(MAKE) clean -C executors/plugins
	$(MAKE) clean -C tests

test-results: $(ALL_RESULTS_TARGETS)

test: mk_go_test ## run unit tests on all go packages

integration-test: ## run venom integration tests declared in tests/ directory
	$(MAKE) start-test-stack -C tests

test-xunit: mk_go_test-xunit ## generate xunit report using the results of previous 'make test', useful on CDS only
	$(MAKE) test-results

package: ## build the docker image with existing binaries from dist directory
	docker build --tag $(DOCKER_IMAGE) .
