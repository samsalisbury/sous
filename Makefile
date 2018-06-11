SHELL := /usr/bin/env bash

POSTGRES_DATA_VOLUME_NAME ?= sous_dev_postgres_data
ifeq ($(shell docker volume ls -q --filter name=^$(POSTGRES_DATA_VOLUME_NAME)$$),)
POSTGRES_DATA_VOLUME_EXISTS := NO
else
POSTGRES_DATA_VOLUME_EXISTS := YES
endif

POSTGRES_CONTAINER_NAME ?= sous_dev_postgres
POSTGRES_CONTAINER_ID = $(shell docker ps -q --no-trunc --filter name=^/$(POSTGRES_CONTAINER_NAME)$$)
POSTGRES_CONTAINER_RUNNING = $(shell if [ "$(POSTGRES_CONTAINER_ID)" = "" ]; then echo NO; else echo YES; fi)

define STOP_POSTGRES
docker ps
@if [ $(POSTGRES_CONTAINER_RUNNING) = YES ]; then docker stop $(POSTGRES_CONTAINER_NAME) && echo Waiting for postgres to stop...; echo Container exited with code $$(docker wait $(POSTGRES_CONTAINER_ID)); echo Postgres container stopped: $(POSTGRES_CONTAINER_NAME); docker rm $(POSTGRES_CONTAINER_ID) && echo Used postgres container deleted.; else echo Postgres container not running.; fi
endef

define START_POSTGRES
endef

define DELETE_POSTGRES_DATA
@if [ $(POSTGRES_DATA_VOLUME_EXISTS) = YES ]; then docker volume rm $(POSTGRES_DATA_VOLUME_NAME) && echo Postgres data volume deleted: $(POSTGRES_DATA_VOLUME_NAME); else echo Postgres data volume does not exist.; fi
endef

SOUS_TERSE_LOGGING ?= YES

XDG_DATA_HOME ?= $(HOME)/.local/share
DEV_POSTGRES_DIR ?= $(XDG_DATA_HOME)/sous/postgres_docker
DEV_POSTGRES_DATA_DIR ?= $(DEV_POSTGRES_DIR)/data
PGPORT ?= 6543
USER_ID ?= $(shell id -u)
GROUP_ID ?= $(shell id -g)

LOCAL_SERVER_LISTEN ?= localhost:7771

DOCKER_HOST_IP_PARSED ?= $(shell echo "$(DOCKER_HOST)" | grep -E -o '(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)')
DOCKER_HOST_LOCALHOST := localhost
DOCKER_HOST_IP := $(if $(DOCKER_HOST_IP_PARSED),$(DOCKER_HOST_IP_PARSED),$(DOCKER_HOST_LOCALHOST))

PGHOST := $(DOCKER_HOST_IP)


DB_NAME = sous
TEST_DB_NAME = sous_test_template
LIQUIBASE_SERVER := jdbc:postgresql://localhost:$(PGPORT)

LIQUIBASE_FLAGS := $(LIQUIBASE_SERVER)/$(DB_NAME)?user=postgres
LIQUIBASE_DEV_FLAGS := $(LIQUIBASE_SERVER)/sous?user=postgres
LIQUIBASE_TEST_FLAGS := $(LIQUIBASE_SERVER)/$(TEST_DB_NAME)?user=postgres

GO_VERSION := 1.10
DESCRIPTION := "Sous is a tool for building, testing, and deploying applications, using Docker, Mesos, and Singularity."
URL := https://github.com/opentable/sous

ifneq ($(GO_TEST_RUN),)
EXTRA_GO_TEST_FLAGS := $(EXTRA_GO_TEST_FLAGS) -run $(GO_TEST_RUN)
endif

TAG_TEST := git describe --exact-match --abbrev=0 2>/dev/null
ifeq ($(shell $(TAG_TEST) ; echo $$?), 128)
GIT_TAG := 0.0.0
else
GIT_TAG := $(shell $(TAG_TEST))
endif

# TODO SS: Find out why this is necessary.
# Note: The Darwin test is arbitrary; simply "running on macOS" is probably not the problem,
# but right now this is not necessary on any of the Linux machines in dev or CI.
ifeq ($(shell uname),Darwin)
DESTROY_SINGULARITY_BETWEEN_SMOKE_TEST_CASES ?= YES
else
DESTROY_SINGULARITY_BETWEEN_SMOKE_TEST_CASES ?= NO
endif

REPO_ROOT := $(shell git rev-parse --show-toplevel)
SMOKE_TEST_BASEDIR ?= $(REPO_ROOT)/.smoketest
SMOKE_TEST_DATA_DIR ?= $(SMOKE_TEST_BASEDIR)/$(DATE)
SMOKE_TEST_LATEST_LINK ?= $(SMOKE_TEST_BASEDIR)/latest
SMOKE_TEST_BINARY ?= $(SMOKE_TEST_DATA_DIR)/sous
SMOKE_TEST_TIMEOUT ?= 30m

# install-dev uses DEV_DESC and DATE to make a git described, timestamped dev build.
DEV_DESC ?= $(shell git describe)
ifneq ($(shell echo $(DEV_DESC) | grep -E '^\d+\.\d+\.\d+'),$(DEV_DESC))
DEV_DESC := 0.0.0-$(DEV_DESC)
endif

DATE := $(shell date +%Y-%m-%dT%H-%M-%S)
DEV_VERSION := "$(DEV_DESC)-devbuild-$(DATE)"

SOUS_BIN_PATH := $(shell which sous 2> /dev/null || echo $$GOPATH/bin/sous)

# Sous releases are tagged with format v0.0.0. semv library
# does not understand the v prefix, so this lops it off.
SOUS_VERSION ?= $(shell echo $(GIT_TAG) | sed 's/^v//')

DB_NAME ?= sous

ifeq ($(shell git diff-index --quiet HEAD ; echo $$?),0)
COMMIT := $(shell git rev-parse HEAD)
else
COMMIT := DIRTY
endif

ifndef SOUS_QA_DESC
QA_DESC := `pwd`/qa_desc.json
else
QA_DESC := $(SOUS_QA_DESC)
endif

ifndef INTEGRATION_TEST_TIMEOUT
INTEGRATION_TEST_TIMEOUT := 30m
endif


FLAGS := '-X "main.Revision=$(COMMIT)" -X "main.VersionString=$(SOUS_VERSION)"'
BIN_DIR := artifacts/bin
DARWIN_RELEASE_DIR := sous-darwin-amd64_$(SOUS_VERSION)
LINUX_RELEASE_DIR := sous-linux-amd64_$(SOUS_VERSION)
RELEASE_DIRS := $(DARWIN_RELEASE_DIR) $(LINUX_RELEASE_DIR)
DARWIN_TARBALL := $(DARWIN_RELEASE_DIR).tar.gz
LINUX_TARBALL := $(LINUX_RELEASE_DIR).tar.gz

# RELEASE_FILES are the files that this system can produce for make release.
RELEASE_FILES := artifacts/$(DARWIN_TARBALL) artifacts/$(LINUX_TARBALL)
# Right now only Linux systems can produce the .deb package as fpm fails on
# Darwin...
ifeq ($(shell uname),Linux)
RELEASE_FILES := $(RELEASE_FILES) artifacts/sous_$(SOUS_VERSION)_amd64.deb
endif

COVER_DIR := /tmp/sous-cover
TEST_VERBOSE := $(if $(VERBOSE),-v,)
TEST_TEAMCITY := $(if $(TEAMCITY),| ./dev_support/gotest-to-teamcity)
SOUS_PACKAGES:= $(shell go list -f '{{.ImportPath}}' ./... | grep -v 'vendor')
GO_TEST_PATHS ?= $(shell go list -f '{{if len .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)
SOUS_TC_PACKAGES=$(shell docker run --rm -v $(PWD):/go/src/github.com/opentable/sous -w /go/src/github.com/opentable/sous golang:1.10 go list -f '{{if len .TestGoFiles}}{{.ImportPath}}{{end}}' ./... | sed 's/_\/app/github.com\/opentable\/sous/')

DOCKER_BUILD_RELEASE := docker run --rm -e GOOS=$$GOOS -e GOARCH=$$GOARCH -e OUTPUT_BIN=$$OUTPUT_BIN -v $(PWD):/go/src/github.com/opentable/sous -w /go/src/github.com/opentable/sous golang:1.10 bash -c "( go build -o $$OUTPUT_BIN -ldflags $(FLAGS) && chown $(USER_ID):$(GROUP_ID) $$OUTPUT_BIN )"

FIND_EXCLUSIONS := -type d -name \\.git -prune -o -type d -name \\.smoketest -prune

GO_FILES := $(shell find . $(FIND_EXCLUSIONS) -o -regex '.*\.go')
GO_PROJECT_FILES := $(shell find . $(FIND_EXCLUSIONS) -o -type d -name vendor -prune -o -regex '.*\.go')

SOUS_CONTAINER_IMAGES:= "docker images | egrep '127.0.0.1:5000|testregistry_'"
TC_TEMP_DIR ?= /tmp/sous

print-%  : ; @echo $* = $($*)
export-% : ; @echo $($*)
help:
	@echo --- options:
	@echo make clean
	@echo "make clean-containers: Destroy and delete local testing containers."
	@echo make coverage
	@echo make legendary
	@echo "make release:  Both linux and darwin"
	@echo "make setup-containers: pull and start containers for integration testing."
	@echo "make test: all tests"
	@echo "make test-unit: unit tests"
	@echo "make test-gofmt: gofmt tests"
	@echo "make test-integration: integration tests"
	@echo "make test-staticcheck: runs static code analysis against project packages."
	@echo "make wip: puts a marker file into workspace to prevent Travis from passing the build."
	@echo "make build-debug: builds a linux debug version "
	@echo "make generate-ctags: builds a tags file for project"
	@echo
	@echo "This system can build the following release packages:"
	@echo "$(RELEASE_FILES)"
	@echo "Add VERBOSE=1 for tons of extra output."

.PHONY: print-go-files
print-go-files:
	@echo $(GO_FILES)

.PHONY: build-debug-linux build-debug-darwin
build-debug: build-debug-linux build-debug-darwin

build-debug-linux:
	@if [[ $(SOUS_VERSION) != *"debug" ]]; then echo 'missing debug at the end of semv, please add'; exit -1; fi
	echo "building debug version" $(SOUS_VERSION) "to" $(BIN_DIR)
	mkdir -p $(BIN_DIR)
	export OUTPUT_BIN=artifacts/bin/sous-linux-amd64 GOOS=linux GOARCH=amd64; $(DOCKER_BUILD_RELEASE)
	mv ./artifacts/bin/sous-linux-amd64 ./artifacts/bin/sous-linux-$(SOUS_VERSION)

build-debug-darwin:
	@if [[ $(SOUS_VERSION) != *"debug" ]]; then echo 'missing debug at the end of semv, please add'; exit -1; fi
	echo "building debug version" $(SOUS_VERSION) "to" $(BIN_DIR)
	mkdir -p $(BIN_DIR)
	export OUTPUT_BIN=artifacts/bin/sous-darwin-amd64 GOOS=linux GOARCH=amd64; $(DOCKER_BUILD_RELEASE)
	mv ./artifacts/bin/sous-darwin-amd64 ./artifacts/bin/sous-darwin-$(SOUS_VERSION)

install-debug-linux: build-debug-linux
	rm $(SOUS_BIN_PATH) || true
	cp ./artifacts/bin/sous-linux-$(SOUS_VERSION) $(SOUS_BIN_PATH)
	cp ./artifacts/bin/sous-linux-$(SOUS_VERSION) ./artifacts/bin/sous-$(SOUS_VERSION)
	sous version

install-debug-darwin: build-debug-darwin
	brew uninstall opentable/public/sous || true
	rm $(SOUS_BIN_PATH) || true
	cp ./artifacts/bin/sous-darwin-$(SOUS_VERSION) $(SOUS_BIN_PATH)
	sous version

clean:
	rm -rf $(COVER_DIR)
	git ls-files -z -o --exclude=.cleanprotect --exclude-per-directory=.cleanprotect | xargs -0 rm -rf
	rm -f $(QA_DESC)

clean-containers: clean-container-certs clean-running-containers clean-container-images

clean-container-images:
	@if (( $$("$(SOUS_CONTAINER_IMAGES)" | wc -l) > 0 )); then echo 'found docker images, deleting:'; "$(SOUS_CONTAINER_IMAGES)" | awk '{ print $$3 }' | xargs docker rmi -f; fi

clean-container-certs:
	rm -f ./integration/test-registry/docker-registry/testing.crt

clean-running-containers:
	@if (( $$(docker ps -q | wc -l) > 0 )); then echo 'found running containers, killing:'; docker ps -q | xargs docker kill; fi
	@if (( $$(docker ps -aq | wc -l) > 0 )); then echo 'found container instances, deleting:'; docker ps -aq | xargs docker rm --volumes; fi

.PHONY: stop-qa-env
stop-qa-env: ## Stops and removes all docker-compose containers.
	@echo Stopping QA environment... # Redirect output to /dev/null because it gives confusing output when nothing to do.
	@cd integration/test-registry && docker-compose rm -sf >/dev/null 2>&1 || { echo Failed to stop containers; exit 1; }
	@if [ -f "$(QA_DESC)" ]; then rm -f $(QA_DESC); fi

.PHONY: start-qa-env
start-qa-env: setup-containers


gitlog:
	git log `git describe --abbrev=0`..HEAD

install-dev:
	brew uninstall opentable/public/sous || true
	rm "$$(which sous)" || true
	go install -ldflags "-X main.VersionString=$(DEV_VERSION)"
	echo "Now run 'hash -r && sous version' to make sure you are using the dev version of sous."

homebrew:
	@command -v brew > /dev/null 2>&1 || \
		( echo "$(MAKECMDGOALS) requires homebrew, see https://brew.sh/"; \
		exit 1 )

install-brew: homebrew
	rm "$$(which sous)" || true
	brew uninstall opentable/public/sous || true
	brew install opentable/public/sous
	echo "Now run 'hash -r && sous version' to make sure you are using the homebrew-installed sous."

install-fpm:
	gem install --no-ri --no-rdoc fpm

install-jfrog:
	go get github.com/jfrogdev/jfrog-cli-go/jfrog

install-ggen:
	cd bin/ggen && go install ./

install-stringer:
	go get golang.org/x/tools/cmd/stringer

install-govendor:
	go get github.com/kardianos/govendor

install-engulf:
	go get github.com/nyarly/engulf

install-staticcheck:
	go install -v ./vendor/honnef.co/go/tools/cmd/staticcheck

install-metalinter:
	go get github.com/alecthomas/gometalinter

install-liquibase:
	brew install liquibase

install-linters: install-metalinter
	gometalinter --install > /dev/null

install-gotags:
	go get -u github.com/jstemmer/gotags

install-build-tools: install-govendor install-engulf install-staticcheck

generate-ctags: install-gotags
	gotags -R -f .tags .

release: $(RELEASE_FILES)

artifactory: artifacts/sous_$(SOUS_VERSION)_amd64.deb
	jfrog rt upload -deb trusty/main/amd64 artifacts/sous_$(SOUS_VERSION)_amd64.deb opentable-ppa/pool/sous_$(SOUS_VERSION)_amd64.deb

deb-build: artifacts/sous_$(SOUS_VERSION)_amd64.deb

linux-build: artifacts/$(LINUX_RELEASE_DIR)/sous
	ln -sf ../$< dev_support/sous_linux

semvertagchk:
	@echo "$(SOUS_VERSION)" | egrep ^[0-9]+\.[0-9]+\.[0-9]+

sous-qa-setup: ./dev_support/sous_qa_setup/*.go ./util/test_with_docker/*.go
	go build $(EXTRA_GO_FLAGS) ./dev_support/sous_qa_setup


reject-wip:
	test ! -f workinprogress

wip:
	touch workinprogress
	git add workinprogress
	git commit --squash=HEAD -m "Making WIP" --no-gpg-sign --no-verify


.cadre/coverage.vim: $(COVER_DIR)/count_merged.txt
	legendary --hitlist --limit 20 $@ $<

coverage: $(COVER_DIR)/count_merged.txt

legendary: .cadre/coverage.vim

test: test-gofmt test-staticcheck test-unit test-integration

test-dev: test-gofmt test-staticcheck test-unit-base legendary

test-staticcheck: install-staticcheck
	echo "staticcheck -ignore "$$(cat staticcheck.ignore)" $(SOUS_PACKAGES)"
	@staticcheck -ignore "$$(cat staticcheck.ignore)" $(SOUS_PACKAGES) || (echo "FAIL: staticcheck" && false)
	echo "staticcheck -tags integration -ignore "$$(cat staticcheck.ignore)" github.com/opentable/sous/integration"
	@staticcheck -tags integration -ignore "$$(cat staticcheck.ignore)" github.com/opentable/sous/integration || (echo "FAIL: staticcheck" && false)

test-metalinter: install-linters
	gometalinter --config gometalinter.json ./...

test-gofmt:
	bin/check-gofmt

.PHONY: test-unit-base
test-unit-base: $(COVER_DIR) $(GO_FILES) postgres-start | postgres-clean # | is "order only"
	PGHOST=$(PGHOST) \
	PGPORT=$(PGPORT) \
	go test $(EXTRA_GO_TEST_FLAGS) $(EXTRA_GO_FLAGS) $(TEST_VERBOSE) \
		-covermode=atomic -coverprofile=$(COVER_DIR)/count_merged.txt \
		-timeout 12m -race $(GO_TEST_PATHS) $(TEST_TEAMCITY)

.PHONY: test-unit
test-unit: postgres-clean test-unit-base

$(COVER_DIR)/count_merged.txt: $(COVER_DIR) $(GO_FILES)
	go test \
		-covermode=count -coverprofile=$(COVER_DIR)/count_merged.txt \
		$(GO_TEST_PATHS)

test-integration: setup-containers postgres-start
	@echo
	@echo
	@echo Integration tests timeout in $(INTEGRATION_TEST_TIMEOUT)
	@echo -n Began at:
	@date
	@echo Set INTEGRATION_TEST_TIMEOUT to override.
	@echo
	@echo
	PGHOST=$(PGHOST) \
	PGPORT=$(PGPORT) \
	SOUS_QA_DESC=$(QA_DESC) go test -count 1 -timeout $(INTEGRATION_TEST_TIMEOUT) $(EXTRA_GO_FLAGS)  $(TEST_VERBOSE) $(EXTRA_GO_TEST_FLAGS) ./integration --tags=integration $(TEST_TEAMCITY)
	@date

$(SMOKE_TEST_BINARY):
	go build -o $@ -tags smoke -ldflags "-X main.VersionString=$(DEV_VERSION)"

$(SMOKE_TEST_DATA_DIR):
	mkdir -p $@

$(SMOKE_TEST_LATEST_LINK): $(SMOKE_TEST_DATA_DIR)
	ln -sfn $< $@

.PHONY: test-smoke-compiles
test-smoke-compiles: ## Checks that the smoke tests compile.
	@go test -c -o /dev/null -tags smoke ./test/smoke && echo Smoke tests compiled.

.PHONY: test-smoke-all
test-smoke-all: test-smoke-compiles $(SMOKE_TEST_BINARY) $(SMOKE_TEST_LATEST_LINK) setup-containers postgres-clean-restart
	@echo "Smoke tests running; time out in $(SMOKE_TEST_TIMEOUT)..."
	ulimit -n 2048 && \
	PGHOST=$(PGHOST) \
	PGPORT=$(PGPORT) \
	SOUS_PG_HOST=$(PGHOST) \
	SOUS_PG_PORT=$(PGPORT) \
	SMOKE_TEST_DATA_DIR=$(SMOKE_TEST_DATA_DIR)/data \
	SMOKE_TEST_BINARY=$(SMOKE_TEST_BINARY) \
	SOUS_QA_DESC=$(QA_DESC) \
	DESTROY_SINGULARITY_BETWEEN_SMOKE_TEST_CASES=$(DESTROY_SINGULARITY_BETWEEN_SMOKE_TEST_CASES) \
	SOUS_TERSE_LOGGING=$(SOUS_TERSE_LOGGING) \
	go test $(EXTRA_GO_TEST_FLAGS) -timeout $(SMOKE_TEST_TIMEOUT) -tags smoke -v -count 1 ./test/smoke $(TEST_TEAMCITY)

.PHONY: test-smoke
test-smoke:
	EXCLUDE_KNOWN_FAILING_TESTS=YES $(MAKE) test-smoke-all

.PHONY: docker-is-working
docker-is-working:
	@docker ps > /dev/null || { echo "'docker ps' failed; please ensure it succeeds and try again."; exit 1; } # Only redirects stdout to dev/null so we still see error messages.
	@echo "'docker ps' succeeded"

$(QA_DESC): docker-is-working sous-qa-setup
	./sous_qa_setup --compose-dir ./integration/test-registry/ --out-path=$(QA_DESC)

setup-containers: $(QA_DESC)

test-cli: setup-containers linux-build
	rm -rf integration/raw_shell_output/0*
	@date
	SOUS_QA_DESC=$(QA_DESC) go test $(EXTRA_GO_FLAGS) $(TEST_VERBOSE) -timeout 20m ./integration --tags=commandline


$(COVER_DIR):
	mkdir -p $@

artifacts/$(DARWIN_RELEASE_DIR)/sous:
	mkdir -p artifacts/$(DARWIN_RELEASE_DIR)
	cp -R doc/ artifacts/$(DARWIN_RELEASE_DIR)/doc
	cp README.md artifacts/$(DARWIN_RELEASE_DIR)
	cp LICENSE artifacts/$(DARWIN_RELEASE_DIR)
	mkdir -p $(BIN_DIR)
	export OUTPUT_BIN=$(BIN_DIR)/sous-darwin-10.6-amd64 GOOS=darwin GOARCH=amd64; $(DOCKER_BUILD_RELEASE)
	mv $(BIN_DIR)/sous-darwin-10.6-amd64 $@

artifacts/$(LINUX_RELEASE_DIR)/sous:
	mkdir -p artifacts/$(LINUX_RELEASE_DIR)
	cp -R doc/ artifacts/$(LINUX_RELEASE_DIR)/doc
	cp README.md artifacts/$(LINUX_RELEASE_DIR)
	cp LICENSE artifacts/$(LINUX_RELEASE_DIR)
	mkdir -p $(BIN_DIR)
	export OUTPUT_BIN=$(BIN_DIR)/sous-linux-amd64 GOOS=linux GOARCH=amd64; $(DOCKER_BUILD_RELEASE)
	mv $(BIN_DIR)/sous-linux-amd64 $@

artifacts/$(LINUX_TARBALL): artifacts/$(LINUX_RELEASE_DIR)/sous
	cd artifacts && tar czv $(LINUX_RELEASE_DIR) > $(LINUX_TARBALL)

artifacts/$(DARWIN_TARBALL): artifacts/$(DARWIN_RELEASE_DIR)/sous
	cd artifacts && tar czv $(DARWIN_RELEASE_DIR) > $(DARWIN_TARBALL)

artifacts/sous_$(SOUS_VERSION)_amd64.deb: artifacts/$(LINUX_RELEASE_DIR)/sous
	fpm -s dir -t deb -n sous -v $(SOUS_VERSION) --description $(DESCRIPTION) --url $(URL) artifacts/$(LINUX_RELEASE_DIR)/sous=/usr/bin/sous
	mv sous_$(SOUS_VERSION)_amd64.deb artifacts/

.PHONY: postgres-start
postgres-start: | postgres-stop postgres-clean # "order only" prereqs
	if ! (docker run --net=host postgres:10.3 pg_isready -h $(DOCKER_HOST_IP) -p $(PGPORT)); then \
		docker run -d --name $(POSTGRES_CONTAINER_NAME) -p $(PGPORT):5432 -v $(POSTGRES_DATA_VOLUME_NAME):/var/lib/postgresql/data postgres:10.3 \
		  -c 'max_connections=1000';\
		echo Waiting until Postgres completes booting...;\
		until docker run --net=host postgres:10.3 pg_isready -h $(DOCKER_HOST_IP) -p $(PGPORT); do sleep 1; done;\
		echo Postgres container started;\
	fi;
	docker run --net=host postgres:10.3 createdb -h localhost -p $(PGPORT) -U postgres -w sous || true
	docker run --net=host --rm -e CHANGELOG_FILE=changelog.xml -v $(PWD)/database:/changelogs -e "URL=$(LIQUIBASE_DEV_FLAGS)" jcastillo/liquibase:0.0.7
	docker run --net=host postgres:10.3 createdb -h localhost -p $(PGPORT) -U postgres -w $(TEST_DB_NAME) || true
	docker run --net=host --rm -e CHANGELOG_FILE=changelog.xml -v $(PWD)/database:/changelogs -e "URL=$(LIQUIBASE_TEST_FLAGS)" jcastillo/liquibase:0.0.7

.PHONY: postgres-restart
postgres-restart: postgres-stop postgres-start

.PHONY: postgres-clean-restart
postgres-clean-restart: postgres-start postgres-clean

postgres-stop:
	$(STOP_POSTGRES)

postgres-connect:
	psql -h $(PGHOST) -p $(PGPORT) --username=postgres $(DB_NAME)

postgres-validate-schema:
	liquibase $(LIQUIBASE_FLAGS) validate

postgres-update-schema: postgres-start
	liquibase $(LIQUIBASE_FLAGS) update

postgres-clean: postgres-stop
	$(DELETE_POSTGRES_DATA)

.PHONY: local-server
local-server:
	@if [ -z "$(EMULATE_CLUSTER)" ]; then echo "Please set EMULATE_CLUSTER=<cluster-name>"; exit 1; fi
	@if [ -z "$(SOUS_GDM_REPO)"]; then echo Please set SOUS_GDM_REPO e.g. git@github.com:my-org/my-gdm.git; exit 1; fi
	@echo
	@echo "WARNING WARNING WARNING WARNING WARNING WARNING WARNING"
	@echo "WARNING => Starting local sous server emulating $(EMULATE_CLUSTER)"
	@echo "WARNING => This server can only deploy to $(EMULATE_CLUSTER)"
	@echo "WARNING => This server can read and write to $(SOUS_GDM_REPO)"
	@echo "WARNING => This may cause inconsistent data."
	@echo "WARNING => If you still want to proceed, set SOUS_SERVER=http://$(LOCAL_SERVER_LISTEN)"
	@echo "WARNING => Then your local sous client will run against this server."
	@echo "WARNING WARNING WARNING WARNING WARNING WARNING WARNING"
	@echo
	@export DIR=$(PWD)/.sous-gdm-temp && rm -rf "$$DIR" && git clone $(SOUS_GDM_REPO) $$DIR && SOUS_SIBLING_URLS='{"$(EMULATE_CLUSTER)": "http://$(LOCAL_SERVER_LISTEN)"}' SOUS_STATE_LOCATION=$$DIR SOUS_PG_HOST=$(PGHOST) SOUS_PG_PORT=$(PGPORT) SOUS_PG_USER=postgres sous server -listen $(LOCAL_SERVER_LISTEN) -autoresolver=false -d -v

.PHONY: artifactory clean clean-containers clean-container-certs \
	clean-running-containers clean-container-images coverage deb-build \
	install-fpm install-jfrog install-ggen install-build-tools legendary release \
	semvertagchk test test-gofmt test-integration setup-containers test-unit \
	reject-wip wip staticcheck postgres-start postgres-stop postgres-connect \
	postgres-clean postgres-create-testdb build-debug homebrew install-gotags \
	install-debug-linux install-debug-darwin
