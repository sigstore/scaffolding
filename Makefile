GIT_TAG ?= $(shell git describe --tags --always --dirty)

LDFLAGS=-buildid= -X sigs.k8s.io/release-utils/version.gitVersion=$(GIT_TAG)

KO_DOCKER_REPO ?= ghcr.io/sigstore/scaffolding

.PHONY: ko-resolve
ko-resolve:
	# "Doing ko resolve for config"
	# "Build a big bundle of joy, this also produces SBOMs"
	LDFLAGS="$(LDFLAGS)" \
	ko resolve --tags $(GIT_TAG),latest --base-import-paths --recursive --filename ./config --platform=all --image-refs imagerefs > release.yaml

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

.PHONY: ko-apply-checktree
ko-apply-checktree:
	LDFLAGS="$(LDFLAGS)" \
	ko apply -f ./testdata/config/checktree
