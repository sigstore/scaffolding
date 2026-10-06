GIT_TAG ?= $(shell git describe --tags --always --dirty)
GIT_HASH ?= $(shell git rev-parse HEAD)

LDFLAGS=-buildid= -X sigs.k8s.io/release-utils/version.gitVersion=$(GIT_TAG) -X sigs.k8s.io/release-utils/version.gitCommit=$(GIT_HASH)

KO_DOCKER_REPO ?= ghcr.io/sigstore/scaffolding
KO_PLATFORM ?= all
KO_EXTRA_FLAGS ?=

TRILLIAN_VERSION=$(shell cd hack && go list -m -f '{{ .Version }}' github.com/google/trillian)

OMNIWITNESS_VERSION=$(shell cd hack && go list -m -f '{{ .Version }}' github.com/transparency-dev/witness)

lint:
	go list -f '{{.Dir}}/...' -m | xargs golangci-lint run

tidy:
	./hack/modtidy.sh

# These are the subdirs under config that we'll turn into separate artifacts.
artifacts := ctlog fulcio rekor-tiles tsa tuf

.PHONY: ko-resolve
ko-resolve:
	# "Doing ko resolve for config"
	set -e; for artifact in $(artifacts); do \
		LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
		ko resolve --tags $(GIT_TAG),latest -BRf ./config/$$artifact \
		--platform=$(KO_PLATFORM) $(KO_EXTRA_FLAGS) \
		--image-refs imagerefs-$$artifact > release-$$artifact.yaml; \
	done
	# "Building cloudsqlproxy wrapper"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=$(KO_PLATFORM) $(KO_EXTRA_FLAGS) --tags $(GIT_TAG),latest --image-refs imagerefs-cloudsqlproxy ./tools/cloudsqlproxy/cmd/cloudsqlproxy
	# "Building trillian_log_server"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=$(KO_PLATFORM) $(KO_EXTRA_FLAGS) --tags $(TRILLIAN_VERSION),$(GIT_TAG),latest --image-refs imagerefs-trillian_log_server github.com/google/trillian/cmd/trillian_log_server
	# "Building trillian_log_signer"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=$(KO_PLATFORM) $(KO_EXTRA_FLAGS) --tags $(TRILLIAN_VERSION),$(GIT_TAG),latest --image-refs imagerefs-trillian_log_signer github.com/google/trillian/cmd/trillian_log_signer
	# Building omniwitness
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=$(KO_PLATFORM) $(KO_EXTRA_FLAGS) --tags $(OMNIWITNESS_VERSION),$(GIT_TAG),latest --image-refs imagerefs-gcp_omniwitness github.com/transparency-dev/witness/cmd/gcp/omniwitness

.PHONY: ko-resolve-testdata
ko-resolve-testdata:
	# "Doing ko resolve for testdata"
	# "Build a big bundle of joy, this also produces SBOMs"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko resolve --tags $(GIT_TAG),latest --base-import-paths --recursive --filename ./testdata --platform=$(KO_PLATFORM) $(KO_EXTRA_FLAGS) --image-refs testimagerefs > testrelease.yaml

.PHONY: sign-test-images
sign-test-images:
	GIT_HASH=$(GIT_HASH) GIT_VERSION=$(GIT_TAG) ARTIFACT=testimagerefs ./scripts/sign-release-images.sh

.PHONY: sign-release-images
sign-release-images: sign-test-images
	set -e; for artifact in $(artifacts) cloudsqlproxy trillian_log_server trillian_log_signer gcp_omniwitness; do \
		echo "Signing $$artifact"; \
		GIT_HASH=$(GIT_HASH) GIT_VERSION=$(GIT_TAG) ARTIFACT=imagerefs-$$artifact ./scripts/sign-release-images.sh; \
	done

.PHONY: stamp-release-version
stamp-release-version:
	# "Pinning setup-scaffolding-from-release.sh to the release tag"
	@if echo "$(GIT_TAG)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		sed -i.bak 's/^RELEASE_VERSION=.*/RELEASE_VERSION="$(GIT_TAG)"/' hack/setup-scaffolding-from-release.sh && \
		rm hack/setup-scaffolding-from-release.sh.bak; \
	else \
		echo "Not stamping release version: '$(GIT_TAG)' is not a release tag"; \
	fi

.PHONY: release-images
release-images: ko-resolve ko-resolve-testdata stamp-release-version

### Testing

.PHONY: ko-apply
ko-apply:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/

.PHONY: ko-apply-ctlog
ko-apply-ctlog:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/ctlog

.PHONY: ko-apply-fulcio
ko-apply-fulcio:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/fulcio

.PHONY: ko-apply-rekor
ko-apply-rekor:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/rekor-tiles

.PHONY: ko-apply-tsa
ko-apply-tsa:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/tsa

.PHONY: ko-apply-tuf
ko-apply-tuf:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/tuf

.PHONY: ko-apply-prober
ko-apply-prober:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -BRf ./config/prober

.PHONY: ko-apply-sign-job
ko-apply-sign-job:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -f ./testdata/config/sign-job

.PHONY: ko-apply-verify-job
ko-apply-verify-job:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -f ./testdata/config/verify-job

.PHONY: ko-apply-gettoken
ko-apply-gettoken:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -f ./testdata/config/gettoken

.PHONY: build
build: build-tuf-server build-cloudsqlproxy build-ctlog-createctconfig build-fulcio-createcerts build-getoidctoken build-rekor-createsecret build-trillian-createdb build-trillian-createtree build-trillian-updatetree build-tsa-createcertchain build-tuf-createsecret

.PHONY: build-cloudsqlproxy
build-cloudsqlproxy:
	go build -trimpath ./tools/cloudsqlproxy/cmd/cloudsqlproxy

.PHONY: build-ctlog-createctconfig
build-ctlog-createctconfig:
	go build -trimpath ./tools/ctlog/cmd/ctlog/createctconfig

.PHONY: build-fulcio-createcerts
build-fulcio-createcerts:
	go build -trimpath ./tools/fulcio/cmd/fulcio/createcerts

.PHONY: build-getoidctoken
build-getoidctoken:
	go build -trimpath ./tools/getoidctoken/cmd/getoidctoken

.PHONY: build-rekor-createsecret
build-rekor-createsecret:
	go build -trimpath ./tools/rekor/cmd/rekor/rekor-createsecret

.PHONY: build-trillian-createdb
build-trillian-createdb:
	go build -trimpath ./tools/trillian/cmd/trillian/createdb

.PHONY: build-trillian-createtree
build-trillian-createtree:
	go build -trimpath ./tools/trillian/cmd/trillian/createtree

.PHONY: build-trillian-updatetree
build-trillian-updatetree:
	go build -trimpath ./tools/trillian/cmd/trillian/updatetree

.PHONY: build-tsa-createcertchain
build-tsa-createcertchain:
	go build -trimpath ./tools/tsa/cmd/tsa/createcertchain

.PHONY: build-tuf-createsecret
build-tuf-createsecret:
	go build -trimpath ./tools/tuf/cmd/tuf/createsecret

.PHONY: build-tuf-server
build-tuf-server:
	go build -trimpath ./tools/tuf/cmd/tuf/server
