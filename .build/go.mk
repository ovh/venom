

GO_BUILD 			= CGO_ENABLED=0 go build -installsuffix cgo
GO_LIST 			= env GO111MODULE=on go list
TEST_CMD 			= go test -v -timeout 600s -coverprofile=profile.coverprofile
TEST_C_CMD 			= go test -c -coverprofile=profile.coverprofile
TEST_RUN_ARGS 		= -test.v -test.timeout 600s -test.coverprofile=profile.coverprofile
CURRENT_PACKAGE 	= $(shell $(GO_LIST))
VERSION 			:= $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
TARGET_DIST 		:= ./dist
TARGET_RESULTS 		:= ./results

##### =====> Clean <===== #####

mk_go_clean: # clean target directory
	@rm -rf $(TARGET_DIST)
	@rm -rf $(TARGET_RESULTS)
	@for testfile in `find ./ -name "bin.test"`; do \
		rm $$testfile; \
	done;
	@for TST in `find ./ -name "tests.log"`; do \
		rm $$TST; \
	done;
	@for profile in `find ./ -name "*.coverprofile"`; do \
		rm $$profile; \
	done;

##### =====> Compile <===== #####

IS_TEST                    = $(filter test,$(MAKECMDGOALS))
TARGET_OS                  = $(filter-out $(TARGET_OS_EXCLUDED), $(if ${ENABLE_CROSS_COMPILATION},$(if ${OS},${OS}, $(if $(IS_TEST), $(shell go env GOOS), windows darwin linux openbsd freebsd)),$(shell go env GOOS)))
TARGET_ARCH                = $(if ${ARCH},${ARCH}, $(if $(IS_TEST), $(shell go env GOARCH),amd64 arm 386 arm64 ppc64le))
BINARIES                   = $(addprefix $(TARGET_DIST)/, $(addsuffix .$(OS)-$(ARCH)$(if $(IS_WINDOWS),.exe), $(notdir $(TARGET_NAME))))
OSARCHVALID                := $(shell go tool dist list | grep -v '^darwin/arm'|grep -v '^darwin/386'|grep -v '^windows/386'|grep -v '^windows/arm'|grep -v '^openbsd/arm*'|grep -v '^openbsd/386'|grep -v '^freebsd/arm*'|grep -v '^freebsd/386')
IS_OS_ARCH_VALID           = $(filter $(OS)/$(ARCH),$(OSARCHVALID))
CROSS_COMPILED_BINARIES    = $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(if $(IS_OS_ARCH_VALID), $(BINARIES))))
GOFILES                    := $(call get_recursive_files, '.')

mk_go_build:
	$(info *** mk_go_build)

mk_go_build_plugin: 
	@mkdir -p dist/lib && \
	go build -buildmode=plugin -o dist/lib/$(TARGET_NAME).so

mk_go_build_clean:
	@rm -rf dist

$(CROSS_COMPILED_BINARIES): $(GOFILES) $(TARGET_DIST)
	$(info *** compiling $@)
	@GOOS=$(call get_os_from_binary_file,$@) \
	GOARCH=$(call get_arch_from_binary_file,$@) \
	$(GO_BUILD) $(BUILD_MODE) $(LDFLAGS) -o $@;

##### =====> Compile Tests <===== #####

PKGS     = $(or $(PKG),$(shell $(GO_LIST) ./...))
TESTPKGS = $(shell $(GO_LIST) -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))

TESTPKGS_C_FILE = $(addsuffix /bin.test, $(subst $(CURRENT_PACKAGE),.,$(PKG)))
TESTPKGS_C = $(foreach PKG, $(TESTPKGS), $(TESTPKGS_C_FILE))

$(TESTPKGS_C): #main_test.go
	$(info *** compiling test $@)
	@cd $(dir $@) && $(TEST_C_CMD) -o bin.test .

##### =====> Running Tests <===== #####

TESTPKGS_RESULTS_LOG_FILE = $(addsuffix /tests.log, $(subst $(CURRENT_PACKAGE),.,$(PKG)))
TESTPKGS_RESULTS = $(foreach PKG, $(TESTPKGS), $(TESTPKGS_RESULTS_LOG_FILE))

$(HOME)/.richstyle.yml:
	echo "leaveTestPrefix: true" > $(HOME)/.richstyle.yml

GO_RICHGO = ${GOPATH}/bin/richgo
$(GO_RICHGO): $(HOME)/.richstyle.yml
	go get -u github.com/kyoh86/richgo

EXIT_TESTS := 0
$(TESTPKGS_RESULTS): $(GOFILES) $(TESTPKGS_C) $(GO_RICHGO)
	$(info *** executing tests in $(dir $@))
	@-cd $(dir $@) && ./bin.test $(TEST_RUN_ARGS) | tee tests.log | richgo testfilter ;

GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
$(GO_COV_MERGE):
	go get -u github.com/wadey/gocovmerge

GO_GOJUNIT = ${GOPATH}/bin/go-junit-report
$(GO_GOJUNIT):
	go get -u github.com/jstemmer/go-junit-report

GO_COBERTURA = ${GOPATH}/bin/gocover-cobertura
$(GO_COBERTURA):
	go get -u github.com/t-yuki/gocover-cobertura

mk_go_test: $(GO_COV_MERGE) $(GO_COBERTURA) $(GOFILES) $(TARGET_RESULTS) $(TESTPKGS_RESULTS)# Run tests
	@echo "Generating unit tests coverage..."
	@$(GO_COV_MERGE) `find ./ -name "*.coverprofile"` > $(TARGET_RESULTS)/cover.out
	@$(GO_COBERTURA) < $(TARGET_RESULTS)/cover.out > $(TARGET_RESULTS)/coverage.xml
	@go tool cover -html=$(TARGET_RESULTS)/cover.out -o=$(TARGET_RESULTS)/cover.html
	@NB=$$(grep "^FAIL" `find . -type f -name "tests.log"`|grep -v ':0'|wc -l); echo "tests failed $$NB" && exit $$NB

mk_go_test-xunit: $(GO_GOJUNIT) $(TARGET_RESULTS) # Generate test with xunit report
	@echo "Generating xUnit Report..."
	@for TST in `find . -name "tests.log"`; do \
		if [ -s $$TST ]; then \
			FAILED=`grep -E '(FAIL)+\s([a-z\.\/]*)\s\[build failed\]' $$TST | wc -l`; \
			if [ $$FAILED -gt 0 ]; then \
				echo "Build Failed \t\t\t($$TST)"; \
				echo "Build Failed \t\t\t($$TST)" >>  $(TARGET_RESULTS)/fail; \
			else \
				NO_TESTS=`grep -E '\?+\s+([a-z\.\/]*)\s\[no test files\]' $$TST | wc -l`; \
				if [ $$NO_TESTS -gt 0 ]; then \
					echo "No tests found \t\t\t($$TST)"; \
				else \
					if [ ! -z "${CDS_VERSION}" ]; then \
						echo "Sending $$TST to CDS"; \
						worker upload --tag `echo $$TST | sed 's|./||' | sed 's|./||' | sed 's|/|_|g') | sed 's|_tests.log||'` $(abspath $$TST); \
					fi; \
					echo "Generating xUnit report \t$$TST.tests-results.xml"; \
					cat $$TST | $(GO_GOJUNIT) > $$TST.tests-results.xml; \
				fi; \
			fi; \
		else \
			echo "Ignoring empty file \t\t$$TST"; \
		fi; \
	done; \
	for XML in `find . -name "tests.log.tests-results.xml"`; do \
		if [ "$$XML" =  "./tests.log.tests-results.xml" ]; then \
      		PWD=`pwd`; \
		 	mv $$XML $(TARGET_RESULTS)/`basename $(PWD)`.tests-results.xml; \
		else \
			mv $$XML $(TARGET_RESULTS)/`echo $$XML | sed 's|./||' | sed 's|/|_|g' | sed 's|_tests.log||'`; \
		fi; \
	done; \
	rm -f $(TARGET_RESULTS)/report; \
	for XML in `find . -name "*.tests-results.xml"`; do \
		if [ -s $$XML ]; then \
			if grep -q 'testsuite' $$XML; then \
				echo "Generating report: " $$XML; \
				echo "`xmllint --xpath "//testsuite/@name" $$XML | sed 's/name=//' | sed 's/"//g'`" \
				"`xmllint --xpath "//testsuite/@tests" $$XML | sed 's/tests=//' | sed 's/"//g'` Tests :" \
				"`xmllint --xpath "//testsuite/@errors" $$XML 2>/dev/null | sed 's/errors=//' | sed 's/"//g'` Errors ;"\
				"`xmllint --xpath "//testsuite/@failures" $$XML 2>/dev/null | sed 's/failures=//' | sed 's/"//g'` Failures;" \
				"`xmllint --xpath "//testsuite/@skip" $$XML 2>/dev/null | sed 's/skip=//' | sed 's/"//g'` Skipped;" \
				>> $(TARGET_RESULTS)/report; \
			fi; \
		fi; \
	done; \
	if [ -e $(TARGET_RESULTS)/report ]; then \
		cat $(TARGET_RESULTS)/report; \
	fi; \
	echo "#########################"; \
	for XML in `find . -name "*.tests-results.xml"`; do \
		if [ -s $$XML ]; then \
			if grep -q 'errors' $$XML && grep -q 'testsuite' $$XML; then \
				if [ "`xmllint --xpath "//testsuite/@errors" $$XML | sed 's/errors=//' | sed 's/"//g'`" -gt "0" ]; then  \
					echo "	$$XML : Tests failed";  \
				fi; \
			fi; \
			if grep -q 'failures' $$XML && grep -q 'testsuite' $$XML $$XML; then \
				if [ "`xmllint --xpath "//testsuite/@failures" $$XML | sed 's/failures=//' | sed 's/"//g'`" -gt "0" ]; then  \
					echo "	$$XML : Tests failed";  \
				fi; \
			fi; \
		fi; \
	done; \
	if [ -e $(TARGET_RESULTS)/fail ]; then \
		echo "#########################"; \
		echo "ERROR: Test compilation failure"; \
		cat $(TARGET_RESULTS)/fail; \
		exit 1; \
	fi;

##### =====> lint <===== #####

TMP_DIR = /tmp/ovh/venom

OSNAME=$(shell go env GOOS)
CUR_PATH = $(notdir $(shell pwd))

GOLANGCI_DIR = $(TMP_DIR)/$(CUR_PATH)/golangci-lint
GOLANGCI_TMP_BIN = $(GOLANGCI_DIR)/golangci-lint

GOLANGCI_LINT_VERSION=1.31.0
GOLANGCI_CMD = $(GOLANGCI_TMP_BIN) run --allow-parallel-runners -c .golangci.yml
GOLANGCI_LINT_ARCHIVE = golangci-lint-$(GOLANGCI_LINT_VERSION)-$(OSNAME)-amd64.tar.gz

# Run this on localc machine.
# It downloads a version of golangci-lint and execute it locally.
# duration first time ~6s
# duration second time ~2s
.PHONY: lint
lint: $(GOLANGCI_TMP_BIN)
	$(GOLANGCI_DIR)/$(GOLANGCI_CMD)

# install a local golangci-lint if not found.
$(GOLANGCI_TMP_BIN):
	curl -OL https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/$(GOLANGCI_LINT_ARCHIVE)
	mkdir -p $(GOLANGCI_DIR)/
	tar -xf $(GOLANGCI_LINT_ARCHIVE) --strip-components=1 -C $(GOLANGCI_DIR)/
	chmod +x $(GOLANGCI_TMP_BIN)
	rm -f $(GOLANGCI_LINT_ARCHIVE)

mk_go_lint: $(GOLANG_CI_LINT) # run golangci lint
	$(info *** running lint)
	$(GOLANGCI_CMD)

##### =====> Internals <===== #####

$(TARGET_RESULTS):
	$(info create $(TARGET_RESULTS) directory)
	@mkdir -p $(TARGET_RESULTS)

$(TARGET_DIST):
	$(info create $(TARGET_DIST) directory)
	@mkdir -p $(TARGET_DIST)

define get_os_from_binary_file
$(strip $(shell echo $(1) | cut -d '.' -f 2 | cut -d'-' -f 1))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | cut -d'-' -f 2)))
endef

define get_recursive_files
$(subst ./,,$(shell find $(1) -type f -name "*.go" -print))
endef