#!/usr/bin/env bash

# Copyright 2021 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# Knative sets this to the root of the git repo otherwise and is unhappy
REPO_ROOT_DIR=$(dirname "$0")/..
pushd ${REPO_ROOT_DIR}
echo === Vendoring scripts
go mod vendor

source $(dirname "$0")/../vendor/knative.dev/hack/library.sh

go_update_deps "$@"

echo === Removing vendor/
rm -rf $REPO_ROOT_DIR/vendor/
