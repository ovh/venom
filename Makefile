TARGET_DIR 							:= 	./dist
IS_TEST 							:= 	$(filter test,$(MAKECMDGOALS))
TARGET_OS 							:= 	$(if ${OS},${OS}, $(if $(IS_TEST), $(shell go env GOOS), darwin linux freebsd))
TARGET_ARCH 						:= 	$(if ${ARCH},${ARCH}, $(if $(IS_TEST), $(shell go env GOARCH),amd64))
VERSION 							:= 	$(if ${CDS_SEMVER},${CDS_SEMVER},$(shell git describe)-snapshot)
GITHASH 							:= 	$(if ${GIT_HASH},${GIT_HASH},$(shell git log -1 --format="%H"))
BUILDTIME 							:= 	$(shell date "+%m/%d/%y-%H:%M:%S")
PREFIX								:=	$(if ${PREFIX}, ${PREFIX}, /usr/local)
INSTALL_DIR							:=	$(if ${INSTALL_DIR}, ${INSTALL_DIR}, $(PREFIX)/bin/venom-$(VERSION))
VENOM_DIR							:=	$(if $(VENOM_DIR), $(VENOM_DIR), $(HOME)/.venom.d/)
GO_BUILD 							:=	go build
LDFLAGS 							=	-ldflags "-X github.com/ovh/venom.Version=$(VERSION) -X github.com/ovh/venom.GOOS=$(GOOS) -X github.com/ovh/venom.GOARCH=$(GOARCH) -X github.com/ovh/venom.GITHASH=$(GITHASH) -X github.com/ovh/venom.BUILDTIME=$(BUILDTIME)"
CORE 								:=	cmd/venom
EXECUTORS 							:=	$(shell ls -d executors/*)
IS_WINDOWS							= 	$(filter $(OS),windows)
CORE_BINARIES 						=	$(addprefix $(TARGET_DIR)/, $(addsuffix _$(OS)_$(ARCH)$(if $(IS_WINDOWS),.exe), $(notdir $(CORE))))	
CROSS_COMPILED_CORE_BINARIES 		= 	$(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(CORE_BINARIES)))
EXECUTORS_BINARIES 					= 	$(addprefix $(TARGET_DIR)/$(EXEC)/, $(addsuffix _$(OS)_$(ARCH)$(if $(IS_WINDOWS),.exe), $(patsubst executors/%, %, $(EXEC))))	
CROSS_COMPILED_EXECUTORS_BINARIES 	=	$(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(foreach EXEC, $(EXECUTORS), $(EXECUTORS_BINARIES))))
COMMON_FILES 						=	$(shell ls *.go lib/cmd/*.go)
EXECUTOR_COMMON_FILES				=  	$(shell find lib/executor -type f -name "*.go" -print)
TEST_START_SMTP := docker run -d --name fakesmtp -p 1025:25 -v /tmp/fakemail:/var/mail digiplant/fake-smtp
TEST_KILL_SMTP := docker kill fakesmtp; docker rm fakesmtp || true
TEST_START_KAFKA := docker network create kafka && docker run --net=kafka -d --name=zookeeper -e ZOOKEEPER_CLIENT_PORT=2181 confluentinc/cp-zookeeper:4.1.0 && docker run --net=kafka -d -p 9092:9092 --name=kafka -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092 -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 confluentinc/cp-kafka:4.1.0
TEST_KILL_KAFKA :=  docker kill zookeeper; docker rm zookeeper; docker kill kafka; docker rm kafka; docker network rm kafka || true

define get_recursive_files
$(shell find $(1) -type f -name "*.go" -print)
endef

define get_os_from_binary_file
$(strip $(shell echo $(1) | cut -d'_' -f 2))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | cut -d'_' -f 3)))
endef

define get_executor_path_from_binary_file
$(strip $(shell echo $(1) | cut -d'/' -f2-3))
endef

.PHONY: clean build gobuild gomodclean gomod test install uninstall mklink

clean:
	$(info *** remove $(TARGET_DIR) directory)
	@rm -rf $(TARGET_DIR)

build: $(TARGET_DIR) $(CROSS_COMPILED_EXECUTORS_BINARIES) $(CROSS_COMPILED_CORE_BINARIES)
	$(info *******************************************************)
	$(info *** venom $(VERSION) build successful)

$(TARGET_DIR):
	$(info *** create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)/executors

$(CROSS_COMPILED_CORE_BINARIES): $(COMMON_FILES) $(call get_recursive_files, $(CORE))
	$(info *** building core binary $@)
	@$(MAKE) --no-print-directory  gobuild GOOS=$(call get_os_from_binary_file,$@) GOARCH=$(call get_arch_from_binary_file,$@) OUTPUT=$@ PACKAGE=$(CORE)

$(CROSS_COMPILED_EXECUTORS_BINARIES): $(COMMON_FILES) $(EXECUTOR_COMMON_FILES) $(call get_recursive_files, $(EXECUTORS))
	$(info *** building executor binary $@ from package $(call get_executor_path_from_binary_file,$$@))
	@$(MAKE) --no-print-directory  gobuild GOOS=$(call get_os_from_binary_file,$@) GOARCH=$(call get_arch_from_binary_file,$@) OUTPUT=$@ PACKAGE=$(call get_executor_path_from_binary_file,$@)

gobuild: vendor
	$(info ... Package: $(PACKAGE) OS: $(GOOS) ARCH: $(GOARCH) -> $(OUTPUT))
	@GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD) $(LDFLAGS) -o $(OUTPUT) $(abspath $(PACKAGE))

vendor:
	@$(MAKE) --no-print-directory gomod

gomodclean: vendor
	@echo "removing vendor directory... " && rm -rf vendor
	@echo "cleaning modcache... " && GO111MODULE=on go clean -modcache || true

gomod:
	@echo "running go mod tidy... " && GO111MODULE=on go mod tidy
	@echo "running go mod vendor..." && GO111MODULE=on go mod vendor
	@echo "doing some clean in vendor directory..." && find vendor -type f ! \( -name 'modules.txt' -o -name '*.sum' -o -name '*.mod' -o -name '*.rst' -o -name '*.go' -o -name '*.y' -o -name '*.h' -o -name '*.c' -o -name '*.proto' -o -name '*.tmpl' -o -name '*.s' -o -name '*.pl' \) -exec rm {} \;

test: $(TARGET_DIR) $(CROSS_COMPILED_EXECUTORS_BINARIES) $(CROSS_COMPILED_CORE_BINARIES)
	go test -v ./... -coverprofile coverage.out -coverpkg ./...
	go tool cover -html coverage.out -o coverage.html

integration: $(TARGET_DIR) $(CROSS_COMPILED_EXECUTORS_BINARIES) $(CROSS_COMPILED_CORE_BINARIES)
	$(TEST_KILL_SMTP)
	$(TEST_START_SMTP)
	$(TEST_KILL_KAFKA)
	$(TEST_START_KAFKA)
	$(TARGET_DIR)/venom_$(shell go env GOOS)_$(shell go env GOARCH) run --configuration-dir $(TARGET_DIR)/executors tests/*.yml --log debug

$(INSTALL_DIR)/%:	
	$(info *** create $(INSTALL_DIR) directory)
	@mkdir -p $(INSTALL_DIR)

$(VENOM_DIR):	
	$(info *** create $(VENOM_DIR) directory)
	@mkdir -p $(VENOM_DIR)

uninstall:
	$(info *** removing $(INSTALL_DIR) directory)
	@rm -rf $(INSTALL_DIR)
	$(info *** removing $(VENOM_DIR) directory)
	@rm -rf $(VENOM_DIR) 
	$(info *** removing venom symbolic link)
	@rm -f $(PREFIX)/bin/venom 

install: $(INSTALL_DIR)/venom $(VENOM_DIR) $(TARGET_DIR) $(CROSS_COMPILED_EXECUTORS_BINARIES) $(CROSS_COMPILED_CORE_BINARIES)
	$(info *** moving venom core binaries to $(INSTALL_DIR) ...)
	@mv $(TARGET_DIR)/venom* $(INSTALL_DIR)
	@chmod +x $(INSTALL_DIR)/venom*
	@ln -sf $(INSTALL_DIR)/venom_$(shell go env GOOS)_$(shell go env GOARCH) $(PREFIX)/bin/venom 
	$(info *** moving venom executor binaries to $(VENOM_DIR) ...)
	@mv $(TARGET_DIR)/executors/* $(VENOM_DIR)
	@chmod -R +x $(VENOM_DIR)/*
	@rm -rf $(TARGET_DIR)
	