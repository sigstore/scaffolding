
#
# Copyright 2022 The Sigstore Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TAG ?= $(shell git describe --tags --always --dirty)
KO_DOCKER_REPO ?= ghcr.io/sigstore/scaffolding

.PHONY: ko-resolve
ko-resolve:
	# "Doing ko resolve for config"
	# "Build a big bundle of joy, this also produces SBOMs"
	ko resolve --tags $(GIT_TAG),latest --base-import-paths --recursive --filename ./config --platform=all --image-refs imagerefs > release.yaml

.PHONY: ko-resolve-testdata
ko-resolve-testdata:
	# "Doing ko resolve for testdata"
	# "Build a big bundle of joy, this also produces SBOMs"
	ko resolve --tags $(GIT_TAG),latest --base-import-paths --recursive --filename ./testdata --platform=all --image-refs testimagerefs > testrelease.yaml

imagerefs := $(shell cat imagerefs testimagerefs)
sign-refs := $(foreach ref,$(imagerefs),$(ref))
.PHONY: sign-images
sign-images:
	cosign sign -a GIT_TAG=$(GIT_TAG) -a GIT_HASH=$(GIT_HASH) $(sign-refs)

.PHONY: release-images
release-images: ko-resolve ko-resolve-testdata
