#!/usr/bin/env bash
# Copyright 2022 The Sigstore Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# Knative sets this to the root of the git repo otherwise and is unhappy
REPO_ROOT_DIR=${PWD}/$(dirname "$0")/..
pushd "${REPO_ROOT_DIR}"
# Removed by update-deps
echo "=== Vendoring scripts"
go mod vendor

# shellcheck disable=SC1091
source "$(dirname "$0")/../vendor/knative.dev/hack/codegen-library.sh"

# Make sure our dependencies are up-to-date
"${REPO_ROOT_DIR}"/hack/update-deps.sh
