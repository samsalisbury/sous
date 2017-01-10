UNIT_TEST_DIRS := `ls -d */ | egrep -v "integration|vendor|bin"`
PACKAGE_PATH := github.com/opentable/sous
OS := $(shell uname)
SQLITE_URL := https://sqlite.org/2017/sqlite-autoconf-3160200.tar.gz
GO_VERSION := 1.7.3
SOUS_TAG := $(shell git describe --exact-match --abbrev=0)

ifeq ($(shell git diff-index --quiet HEAD ; echo $$?),0)
COMMIT := $(shell git rev-parse HEAD)
else
COMMIT := DIRTY
endif 

ifdef $(SOUS_TAG)
	SOUS_VERSION := $(SOUS_TAG)
else
	SOUS_VERSION := UNSUPPORTED
endif

FLAGS := "-X 'main.Revision=$(COMMIT)' -X 'main.VersionString=$(SOUS_VERSION)'"
BIN_DIR := artifacts/sous-$(SOUS_VERSION)
CONCAT_XGO_ARGS := -go $(GO_VERSION) -branch master -deps $(SQLITE_URL) --dest $(BIN_DIR) --ldflags $(FLAGS)

clean:
	rm -f sous
	rm -rf artifacts
ctags:
	ctags -R

artifacts/sous-$(SOUS_VERSION):
	mkdir -p $@

vet:
	@for d in ${UNIT_TEST_DIRS}; do go vet ${PACKAGE_PATH}/$$d...; done

$(BIN_DIR)/sous-linux-amd64: testlinux artifacts/sous-$(SOUS_VERSION)
	xgo $(CONCAT_XGO_ARGS) --targets=linux/amd64  ./

$(BIN_DIR)/sous-darwin-10.6-amd64: testlinux artifacts/sous-$(SOUS_VERSION)
	xgo $(CONCAT_XGO_ARGS) --targets=darwin/amd64  ./

release: artifacts/sous-$(SOUS_VERSION).tar.gz
	@echo $(SOUS_VERSION)

artifacts/sous-$(SOUS_VERSION).tar.gz: $(BIN_DIR)/sous-linux-amd64 $(BIN_DIR)/sous-darwin-10.6-amd64
	tar czv $(BIN_DIR) > $@
 
testlinux:
ifeq ($(OS),Linux)
	@echo "Good news, this is a Linux machine."
else
	@echo "Releases must be built in a Linux environment." ; /bin/false
endif


.PHONY: clean ctags vet release testlinux
