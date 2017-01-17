SQLITE_URL := https://sqlite.org/2017/sqlite-autoconf-3160200.tar.gz
GO_VERSION := 1.7.3

TAG_TEST := git describe --exact-match --abbrev=0
ifeq ($(shell $(TAG_TEST) ; echo $$?), 128)
GIT_TAG := 0.0.0
else
GIT_TAG := $(shell $(TAG_TEST))
endif

# Sous releases are tagged with format v0.0.0. semv library
# does not understand the v prefix, so this lops it off.
SOUS_VERSION := $(shell echo $(GIT_TAG) | sed 's/^v//')

ifeq ($(shell git diff-index --quiet HEAD ; echo $$?),0)
COMMIT := $(shell git rev-parse HEAD)
else
COMMIT := DIRTY
endif 

FLAGS := "-X 'main.Revision=$(COMMIT)' -X 'main.VersionString=$(SOUS_VERSION)'"
BIN_DIR := artifacts/bin
DARWIN_RELEASE_DIR := sous-darwin-amd64_$(SOUS_VERSION)
LINUX_RELEASE_DIR := sous-linux-amd64_$(SOUS_VERSION)
RELEASE_DIRS := $(DARWIN_RELEASE_DIR) $(LINUX_RELEASE_DIR)
DARWIN_TARBALL := $(DARWIN_RELEASE_DIR).tar.gz
LINUX_TARBALL := $(LINUX_RELEASE_DIR).tar.gz
CONCAT_XGO_ARGS := -go $(GO_VERSION) -branch master -deps $(SQLITE_URL) --dest $(BIN_DIR) --ldflags $(FLAGS)
COVER_DIR := /tmp/sous-cover

clean:
	rm -f sous
	rm -rf artifacts
	rm -rf $(COVER_DIR)
	rm -f bin/ggen/ggen

cover: $(COVER_DIR)
	engulf -s --coverdir=$(COVER_DIR) \
		--exclude '/vendor,\
			integration/?,\
		 	/bin/?,\
			/dev_support/?,\
			/util/test_with_docker/?,\
			/examples/?,\
			/util/cmdr/cmdr-example/?'\
		--exclude-files='raw_client.go$$, _generated.go$$'\
		--merge-base=_merged.txt ./...

gitlog:
	git log `git describe --abbrev=0`..HEAD

install-ggen:
	cd bin/ggen && go install ./

legendary: cover
	legendary --hitlist .cadre/coverage.vim /tmp/sous-cover/*_merged.txt

release: artifacts/$(DARWIN_TARBALL) artifacts/$(LINUX_TARBALL)

semvertagchk:
	@echo "$(SOUS_VERSION)" | egrep ^[0-9]+\.[0-9]+\.[0-9]+	

$(BIN_DIR):
	mkdir -p $@

$(COVER_DIR):
	mkdir -p $@

$(RELEASE_DIRS):
	mkdir -p artifacts/$@
	cp -R doc/ artifacts/$@/doc
	cp README.md artifacts/$@
	cp LICENSE artifacts/$@

artifacts/$(DARWIN_RELEASE_DIR)/sous: $(DARWIN_RELEASE_DIR) $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=darwin/amd64  ./
	mv $(BIN_DIR)/sous-darwin-10.6-amd64 $@

artifacts/$(LINUX_RELEASE_DIR)/sous: $(LINUX_RELEASE_DIR) $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=linux/amd64  ./
	mv $(BIN_DIR)/sous-linux-amd64 $@

artifacts/$(LINUX_TARBALL): artifacts/$(LINUX_RELEASE_DIR)/sous
	cd artifacts && tar czv $(LINUX_RELEASE_DIR) > $(LINUX_TARBALL)

artifacts/$(DARWIN_TARBALL): artifacts/$(DARWIN_RELEASE_DIR)/sous
	cd artifacts && tar czv $(DARWIN_RELEASE_DIR) > $(DARWIN_TARBALL)

.PHONY: clean install-ggen legendary release semvertagchk
