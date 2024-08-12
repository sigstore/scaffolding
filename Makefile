GIT_TAG ?= $(shell git describe --tags --always --dirty)
GIT_HASH ?= $(shell git rev-parse HEAD)

LDFLAGS=-buildid= -X sigs.k8s.io/release-utils/version.gitVersion=$(GIT_TAG) -X sigs.k8s.io/release-utils/version.gitCommit=$(GIT_HASH)

KO_DOCKER_REPO ?= ghcr.io/sigstore/scaffolding

TRILLIAN_VERSION=$(shell go list -m -f '{{ .Version }}' github.com/google/trillian)

# These are the subdirs under config that we'll turn into separate artifacts.
artifacts := trillian ctlog fulcio rekor tsa tuf prober

.PHONY: ko-resolve
ko-resolve:
	# "Doing ko resolve for config"
	$(foreach artifact, $(artifacts), $(shell export LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO); \
	ko resolve --tags $(GIT_TAG),latest -BRf ./config/$(artifact) \
	--platform=all \
	--image-refs imagerefs-$(artifact) > release-$(artifact).yaml )) \
	# "Building cloudsqlproxy wrapper"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=all --tags $(GIT_TAG),latest --image-refs imagerefs-cloudsqlproxy ./cmd/cloudsqlproxy
	# "Building trillian_log_server"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=all --tags $(TRILLIAN_VERSION),$(GIT_TAG),latest --image-refs imagerefs-trillian_log_server github.com/google/trillian/cmd/trillian_log_server
	# "Building trillian_log_signer"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko build --base-import-paths --platform=all --tags $(TRILLIAN_VERSION),$(GIT_TAG),latest --image-refs imagerefs-trillian_log_signer github.com/google/trillian/cmd/trillian_log_signer

.PHONY: ko-resolve-testdata
ko-resolve-testdata:
	# "Doing ko resolve for testdata"
	# "Build a big bundle of joy, this also produces SBOMs"
	LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO) \
	ko resolve --tags $(GIT_TAG),latest --base-import-paths --recursive --filename ./testdata --platform=all --image-refs testimagerefs > testrelease.yaml

.PHONY: sign-test-images
sign-test-images:
	GIT_HASH=$(GIT_HASH) GIT_VERSION=$(GIT_TAG) ARTIFACT=testimagerefs ./scripts/sign-release-images.sh

.PHONY: sign-release-images
sign-release-images: sign-test-images
	$(foreach artifact,$(artifacts), \
		echo "Signing $(artifact)"; export GIT_HASH=$(GIT_HASH) GIT_VERSION=$(GIT_TAG) ARTIFACT=imagerefs-$(artifact); ./scripts/sign-release-images.sh \
	)
	echo "Signing cloudsqlproxy"; export GIT_HASH=$(GIT_HASH) GIT_VERSION=$(GIT_TAG) ARTIFACT=imagerefs-cloudsqlproxy; ./scripts/sign-release-images.sh \

.PHONY: release-images
release-images: ko-resolve ko-resolve-testdata

.PHONY: prober
prober:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/prober

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
	ko apply -BRf ./config/rekor

.PHONY: ko-apply-trillian
ko-apply-trillian:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -v -BRf ./config/trillian

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
build: build-tuf-server build-cloudsqlproxy build-ctlog-createctconfig build-ctlog-managectroots build-ctlog-verifyfulcio build-fulcio-createcerts build-getoidctoken build-rekor-createsecret build-trillian-createdb build-trillian-createtree build-trillian-updatetree build-tsa-createcertchain build-tuf-createsecret

.PHONY: build-cloudsqlproxy
build-cloudsqlproxy:
	go build -trimpath ./cmd/cloudsqlproxy

.PHONY: build-ctlog-createctconfig
build-ctlog-createctconfig:
	go build -trimpath ./cmd/ctlog/createctconfig

.PHONY: build-ctlog-managectroots
build-ctlog-managectroots:
	go build -trimpath ./cmd/ctlog/managectroots

.PHONY: build-ctlog-verifyfulcio
build-ctlog-verifyfulcio:
	go build -trimpath ./cmd/ctlog/verifyfulcio

.PHONY: build-fulcio-createcerts
build-fulcio-createcerts:
	go build -trimpath ./cmd/fulcio/createcerts

.PHONY: build-getoidctoken
build-getoidctoken:
	go build -trimpath ./cmd/getoidctoken

.PHONY: build-rekor-createsecret
build-rekor-createsecret:
	go build -trimpath ./cmd/rekor/rekor-createsecret

.PHONY: build-trillian-createdb
build-trillian-createdb:
	go build -trimpath ./cmd/trillian/createdb

.PHONY: build-trillian-createtree
build-trillian-createtree:
	go build -trimpath ./cmd/trillian/createtree

.PHONY: build-trillian-updatetree
build-trillian-updatetree:
	go build -trimpath ./cmd/trillian/updatetree

.PHONY: build-tsa-createcertchain
build-tsa-createcertchain:
	go build -trimpath ./cmd/tsa/createcertchain

.PHONY: build-tuf-createsecret
build-tuf-createsecret:
	go build -trimpath ./cmd/tuf/createsecret

.PHONY: build-tuf-server
build-tuf-server:
	go build -trimpath ./cmd/tuf/server
