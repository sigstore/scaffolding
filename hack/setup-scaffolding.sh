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

# Because setting up tuf requires peeking into several namespaces, we can't
# do that simply from a Job because a pod can't read secrets in the other
# namespaces that we need.
# Alternative would be to create a full blown controller and reconcile the
# secret from there, which might be something that we might consider doing
# anyways.
# But for now, trying to make forward progress here, we have to run this
# script _after_ creating the scaffolding and once it's up.

# So, after [this](https://github.com/sigstore/scaffolding/blob/main/getting-started.md#then-wait-for-the-jobs-that-setup-dependencies-to-finish) step completes, run this script.

# First create the job which will combine a tuf-system/tuf-secrets secret
# that holds all the information that the tuf server will need to create
# the tuf store.
ko apply -f ./config/tuf/createsecret

# Then wait for it to do it's job
kubectl wait --timeout=5m -n tuf-system --for=condition=Complete jobs createsecret

# Copy the necessary secrets to the tuf-system namespace
# TODO(vaikas): Just copy the bits we care about
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n fulcio-system get secrets fulcio-secret -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -

# Then launch the tuf server
ko apply -BRf ./config/tuf/server
