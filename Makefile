include scripts/dqlite/Makefile

PROJECT := github.com/SimonRichardson/juju-dqlite-backstop
PROJECT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

# GIT_COMMIT the current git commit of this repository
GIT_COMMIT = $(shell git -C $(PROJECT_DIR) rev-parse HEAD 2>/dev/null)

# If .git directory is missing, we are building out of an archive, otherwise report
# if the tree that is checked out is dirty (modified) or clean.
GIT_TREE_STATE = $(if $(shell git -C $(PROJECT_DIR) rev-parse --is-inside-work-tree 2>/dev/null | grep -e 'true'),$(if $(shell git -C $(PROJECT_DIR) status --porcelain),dirty,clean),archive)

define link_flags_version
	-X $(PROJECT)/version.GitCommit=$(GIT_COMMIT) \
	-X $(PROJECT)/version.GitTreeState=$(GIT_TREE_STATE)
endef

# Build tags passed to go install/build.
# Passing no-dqlite will disable building with dqlite.
# Example: BUILD_TAGS="minimal provider_kubernetes"
BUILD_TAGS ?= 

# EXTRA_BUILD_TAGS is not passed in, but built up from context.
EXTRA_BUILD_TAGS =
ifeq (,$(findstring no-dqlite,$(BUILD_TAGS)))
EXTRA_BUILD_TAGS += libsqlite3
EXTRA_BUILD_TAGS += dqlite
endif

# FINAL_BUILD_TAGS is the final list of build tags.
FINAL_BUILD_TAGS=$(shell echo "$(BUILD_TAGS) $(EXTRA_BUILD_TAGS)" | awk '{$$1=$$1};1' | tr ' ' ',')

build: musl-install-if-missing dqlite-install-if-missing
	$(eval OS = $(shell go env GOOS))
	$(eval ARCH = $(shell go env GOARCH))
	env PATH="${MUSL_BIN_PATH}:${PATH}" \
		CC="musl-gcc" \
		CGO_CFLAGS="-I${DQLITE_EXTRACTED_DEPS_ARCHIVE_PATH}/include" \
		CGO_LDFLAGS="-L${DQLITE_EXTRACTED_DEPS_ARCHIVE_PATH} -luv -lraft -ldqlite -llz4 -lsqlite3" \
		CGO_LDFLAGS_ALLOW="(-Wl,-wrap,pthread_create)|(-Wl,-z,now)" \
		LD_LIBRARY_PATH="${DQLITE_EXTRACTED_DEPS_ARCHIVE_PATH}" \
		CGO_ENABLED=1 \
		GOOS=${OS} \
		GOARCH=${BUILD_ARCH} \
		go build \
			-tags=$(FINAL_BUILD_TAGS) \
			-o bin/juju-dqlite-backstop \
			-ldflags "-s -w -linkmode 'external' -extldflags '-static' $(link_flags_version)" \
			./cmd/juju-dqlite-backstop

clean:
	rm -rf bin/*

.PHONY: build clean
