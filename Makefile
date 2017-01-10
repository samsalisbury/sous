SQLITE_URL := https://sqlite.org/2017/sqlite-autoconf-3160200.tar.gz
GO_VERSION := 1.7.3

TAG_TEST := git describe --exact-match --abbrev=0
ifeq ($(shell $(TAG_TEST) ; echo $$?), 128)
GIT_TAG := UNSUPPORTED
else
GIT_TAG := $(shell $(TAG_TEST))
endif

ifeq ($(shell git diff-index --quiet HEAD ; echo $$?),0)
COMMIT := $(shell git rev-parse HEAD)
else
COMMIT := DIRTY
endif 

FLAGS := "-X 'main.Revision=$(COMMIT)' -X 'main.VersionString=$(GIT_TAG)'"
BIN_DIR := artifacts/sous-$(GIT_TAG)
CONCAT_XGO_ARGS := -go $(GO_VERSION) -branch master -deps $(SQLITE_URL) --dest $(BIN_DIR) --ldflags $(FLAGS)

clean:
	rm -f sous
	rm -rf artifacts

release: artifacts/sous-$(GIT_TAG).tar.gz

$(BIN_DIR):
	mkdir -p $@
	cp -R doc/ $@
	cp README.md $@
	cp LICENSE $@

artifacts/sous-$(GIT_TAG).tar.gz: binaries
	tar czv $(BIN_DIR) > $@
 
binaries: $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=linux/amd64,darwin/amd64  ./


.PHONY: binaries clean release
