#!/usr/bin/env bash

# Copyright 2021 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Knative sets this to the root of the git repo otherwise and is unhappy
REPO_ROOT_DIR=$PWD/$(dirname "$0")/..
pushd ${REPO_ROOT_DIR}
# Removed by update-deps
echo === Vendoring scripts
go mod vendor

source $(dirname $0)/../vendor/knative.dev/hack/codegen-library.sh

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/hack/update-deps.sh
