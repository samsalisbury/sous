SQLITE_URL := https://sqlite.org/2017/sqlite-autoconf-3160200.tar.gz
GO_VERSION := 1.7.3

TAG_TEST := git describe --exact-match --abbrev=0
ifeq ($(shell $(TAG_TEST) ; echo $$?), 128)
GIT_TAG := v0.0.0
else
GIT_TAG := $(shell $(TAG_TEST))
endif

ifeq ($(shell git diff-index --quiet HEAD ; echo $$?),0)
COMMIT := $(shell git rev-parse HEAD)
else
COMMIT := DIRTY
endif 

FLAGS := "-X 'main.Revision=$(COMMIT)' -X 'main.VersionString=$(GIT_TAG)'"
BIN_DIR := artifacts/bin
DARWIN_RELEASE_DIR := artifacts/sous-darwin-amd64_$(GIT_TAG)
LINUX_RELEASE_DIR := artifacts/sous-linux-amd64_$(GIT_TAG)
RELEASE_DIRS := $(DARWIN_RELEASE_DIR) $(LINUX_RELEASE_DIR)
DARWIN_TARBALL := $(DARWIN_RELEASE_DIR).tar.gz
LINUX_TARBALL := $(LINUX_RELEASE_DIR).tar.gz
CONCAT_XGO_ARGS := -go $(GO_VERSION) -branch master -deps $(SQLITE_URL) --dest $(BIN_DIR) --ldflags $(FLAGS)

clean:
	rm -f sous
	rm -rf artifacts

release: $(DARWIN_TARBALL) $(LINUX_TARBALL)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(RELEASE_DIRS):
	mkdir -p $@
	cp -R doc/ $@/doc
	cp README.md $@
	cp LICENSE $@

$(DARWIN_RELEASE_DIR)/sous: $(DARWIN_RELEASE_DIR) $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=darwin/amd64  ./
	mv $(BIN_DIR)/sous-darwin-10.6-amd64 $@

$(LINUX_RELEASE_DIR)/sous: $(LINUX_RELEASE_DIR) $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=linux/amd64  ./
	mv $(BIN_DIR)/sous-linux-amd64 $@

$(LINUX_TARBALL): $(LINUX_RELEASE_DIR)/sous
	tar czv $(LINUX_RELEASE_DIR) > $@

$(DARWIN_TARBALL): $(DARWIN_RELEASE_DIR)/sous
	tar czv $(DARWIN_RELEASE_DIR) > $@

.PHONY: clean release
