SQLITE_URL := https://sqlite.org/2017/sqlite-autoconf-3160200.tar.gz
GO_VERSION := 1.7.3

TAG_TEST := git describe --exact-match --abbrev=0
ifeq ($(shell $(TAG_TEST) ; echo $$?), 128)
GIT_TAG := v0.0.0
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
BIN_DIR := artifacts/sous-$(SOUS_VERSION)
CONCAT_XGO_ARGS := -go $(GO_VERSION) -branch master -deps $(SQLITE_URL) --dest $(BIN_DIR) --ldflags $(FLAGS)

clean:
	rm -f sous
	rm -rf artifacts

release: artifacts/sous-v$(SOUS_VERSION)-darwin-10.6-amd64.tar.gz artifacts/sous-v$(SOUS_VERSION)-linux-amd64.tar.gz

$(BIN_DIR):
	mkdir -p $@
	cp -R doc/ $@/doc
	cp README.md $@
	cp LICENSE $@

artifacts/sous-v$(SOUS_VERSION)-darwin-10.6-amd64.tar.gz: binaries
	cd $(BIN_DIR) && tar czv \
		--exclude 'sous-linux-amd64' \
		--transform 's|sous-darwin-10.6-amd64|sous|' \
		. > ../../$@

artifacts/sous-v$(SOUS_VERSION)-linux-amd64.tar.gz: binaries
	cd $(BIN_DIR) && tar czv \
		--exclude 'sous-darwin-10.6-amd64' \
		--transform 's|sous-linux-amd64|sous|' \
		. > ../../$@
 
binaries: $(BIN_DIR)
	xgo $(CONCAT_XGO_ARGS) --targets=linux/amd64,darwin/amd64  ./


.PHONY: binaries clean release
