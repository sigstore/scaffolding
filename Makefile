GIT_TAG ?= $(shell git describe --tags --always --dirty)

LDFLAGS=-buildid= -X sigs.k8s.io/release-utils/version.gitVersion=$(GIT_TAG)

KO_DOCKER_REPO ?= ghcr.io/sigstore/scaffolding

# These are the subdirs under config that we'll turn into separate artifacts.
artifacts := trillian ctlog fulcio rekor tuf

.PHONY: ko-resolve
ko-resolve:
	# "Doing ko resolve for config"
	$(foreach artifact, $(artifacts), $(shell export LDFLAGS="$(LDFLAGS)"; \
	ko resolve --tags $(GIT_TAG),latest -BRf ./config/$(artifact) \
	--platform=all \
	--image-refs imagerefs-$(artifact) > release-$(artifact).yaml )) \

	# Then collect all the imagerefs from various imageref-* produced above
	# because otherwise they would stomp on each other above if writing to same
	# file.
	$(foreach artifact, $(artifacts), $(shell cat imagerefs-$(artifact) >> ./imagerefs )) \

.PHONY: ko-resolve-testdata
ko-resolve-testdata:
	# "Doing ko resolve for testdata"
	# "Build a big bundle of joy, this also produces SBOMs"
	LDFLAGS="$(LDFLAGS)" \
	ko resolve --tags $(GIT_TAG),latest --base-import-paths --recursive --filename ./testdata --platform=all --image-refs testimagerefs > testrelease.yaml

imagerefs := $(shell cat imagerefs testimagerefs)
sign-refs := $(foreach ref,$(imagerefs),$(ref))
.PHONY: sign-images
sign-images:
	cosign sign -a GIT_TAG=$(GIT_TAG) -a GIT_HASH=$(GIT_HASH) $(sign-refs)

.PHONY: release-images
release-images: ko-resolve ko-resolve-testdata

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
