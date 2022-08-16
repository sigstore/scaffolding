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

# Default
RELEASE_VERSION="v0.4.3"

while [[ $# -ne 0 ]]; do
  parameter="$1"
  case "${parameter}" in
    --release-version)
      shift
      RELEASE_VERSION="$1"
      ;;
    *) echo "unknown option ${parameter}"; exit 1 ;;
  esac
  shift
done

echo "Installing release version: $RELEASE_VERSION"
TRILLIAN=https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-trillian.yaml
REKOR=https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-rekor.yaml
FULCIO=https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-fulcio.yaml
CTLOG=https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-ctlog.yaml
TUF=https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-tuf.yaml

# Since the behaviour on oidc is different on k8s <1.23, check to see if we
# need to do some mucking with the Fulcio config
NEED_TO_UPDATE_FULCIO_CONFIG="false"
K8S_SERVER_VERSION=$(kubectl version -ojson | yq '.serverVersion.minor' -)

if [ "${K8S_SERVER_VERSION}" == "21" ] || [ "${K8S_SERVER_VERSION}" == "22" ]; then
  echo "Running on k8s 1.${K8S_SERVER_VERSION}.x will update Fulcio accordingly"
  NEED_TO_UPDATE_FULCIO_CONFIG="true"
fi

# Install Trillian and wait for it to come up
echo '::group:: Install Trillian'
kubectl apply -f "${TRILLIAN}"
echo '::endgroup::'

echo '::group:: Wait for Trillian ready'
kubectl wait --timeout 5m -n trillian-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n trillian-system --for=condition=Ready ksvc log-server
kubectl wait --timeout 2m -n trillian-system --for=condition=Ready ksvc log-signer
echo '::endgroup::'

# Install Rekor and wait for it to come up
echo '::group:: Install Rekor'
kubectl apply -f "${REKOR}"
echo '::endgroup::'

echo '::group:: Wait for Rekor ready'
kubectl wait --timeout 5m -n rekor-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n rekor-system --for=condition=Ready ksvc rekor
echo '::endgroup::'

# Install Fulcio and wait for it to come up
echo '::group:: Install Fulcio'
if [ "${NEED_TO_UPDATE_FULCIO_CONFIG}" == "true" ]; then
  echo "Fixing Fulcio config for < 1.23.X Kubernetes"
  curl -Ls "${FULCIO}" | sed 's@https://kubernetes.default.svc.cluster.local@https://kubernetes.default.svc@' | kubectl apply -f -
else
  kubectl apply -f "${FULCIO}"
fi

echo '::group:: Wait for Fulcio ready'
kubectl wait --timeout 5m -n fulcio-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n fulcio-system --for=condition=Ready ksvc fulcio
echo '::endgroup::'

# Install CTlog and wait for it to come up
echo '::group:: Install CTLog'
kubectl apply -f "${CTLOG}"
echo '::endgroup::'

echo '::group:: Wait for CTLog ready'
kubectl wait --timeout 5m -n ctlog-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n ctlog-system --for=condition=Ready ksvc ctlog
echo '::endgroup::'

# Install tuf
echo '::group:: Install TUF'
kubectl apply -f "${TUF}"

# Then copy the secrets (even though it's all public stuff, certs, public keys)
# to the tuf-system namespace so that we can construct a tuf root out of it.
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n fulcio-system get secrets fulcio-pub-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n rekor-system get secrets rekor-pub-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
echo '::endgroup::'

# Make sure the tuf jobs complete
echo '::group:: Wait for TUF ready'
kubectl wait --timeout 4m -n tuf-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n tuf-system --for=condition=Ready ksvc tuf
echo '::endgroup::'

# Grab the trusted root
kubectl -n tuf-system get secrets tuf-root -ojsonpath='{.data.root}' | base64 -d > ./root.json

echo "tuf root installed into ./root.json"

# Get the endpoints for various services and expose them
# as env vars.
REKOR_URL=$(kubectl -n rekor-system get ksvc rekor -ojsonpath='{.status.url}')
export REKOR_URL
FULCIO_URL=$(kubectl -n fulcio-system get ksvc fulcio -ojsonpath='{.status.url}')
export FULCIO_URL
CTLOG_URL=$(kubectl -n ctlog-system get ksvc ctlog -ojsonpath='{.status.url}')
export CTLOG_URL
TUF_MIRROR=$(kubectl -n tuf-system get ksvc tuf -ojsonpath='{.status.url}')
export TUF_MIRROR
