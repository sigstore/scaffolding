GIT_TAG ?= $(shell git describe --tags --always --dirty)
GIT_HASH ?= $(shell git rev-parse HEAD)

LDFLAGS=-buildid= -X sigs.k8s.io/release-utils/version.gitVersion=$(GIT_TAG)

KO_DOCKER_REPO ?= ghcr.io/sigstore/scaffolding

# These are the subdirs under config that we'll turn into separate artifacts.
artifacts := trillian ctlog fulcio rekor tsa tuf prober

.PHONY: ko-resolve
ko-resolve:
	# "Doing ko resolve for config"
	$(foreach artifact, $(artifacts), $(shell export LDFLAGS="$(LDFLAGS)" KO_DOCKER_REPO=$(KO_DOCKER_REPO); \
	ko resolve --tags $(GIT_TAG),latest -BRf ./config/$(artifact) \
	--platform=all \
	--image-refs imagerefs-$(artifact) > release-$(artifact).yaml )) \

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
	ko apply -BRf ./config/trillian

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
