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
make ko-apply-trillian
echo '::endgroup::'

echo '::group:: Wait for Trillian ready'
kubectl wait --timeout 2m -n trillian-system --for=condition=Ready ksvc log-server
kubectl wait --timeout 2m -n trillian-system --for=condition=Ready ksvc log-signer
echo '::endgroup::'

# Install Rekor and wait for it to come up
echo '::group:: Install Rekor'
make ko-apply-rekor
echo '::endgroup::'

echo '::group:: Wait for Rekor ready'
kubectl wait --timeout 5m -n rekor-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n rekor-system --for=condition=Ready ksvc rekor
echo '::endgroup::'

# Install Fulcio and wait for it to come up
echo '::group:: Install Fulcio'
if [ "${NEED_TO_UPDATE_FULCIO_CONFIG}" == "true" ]; then
  echo "Fixing Fulcio config"
  cp config/fulcio/fulcio/200-configmap.yaml ./200-configmap.yaml
  # The sed works differently in mac and other places, so just shuffle
  # files around for now.
  sed 's@https://kubernetes.default.svc.cluster.local@https://kubernetes.default.svc@' config/fulcio/fulcio/200-configmap.yaml > ./200-configmap-new.yaml
  mv ./200-configmap-new.yaml config/fulcio/fulcio/200-configmap.yaml
fi
make ko-apply-fulcio
echo '::endgroup::'

if [ "${NEED_TO_UPDATE_FULCIO_CONFIG}" == "true" ]; then
  echo "Restoring Fulcio config"
  mv ./200-configmap.yaml config/fulcio/fulcio/200-configmap.yaml
fi
echo '::group:: Wait for Fulcio ready'
kubectl wait --timeout 5m -n fulcio-system --for=condition=Complete jobs --all
kubectl wait --timeout 5m -n fulcio-system --for=condition=Ready ksvc fulcio
kubectl wait --timeout 5m -n fulcio-system --for=condition=Ready ksvc fulcio-grpc
echo '::endgroup::'

# Install CTlog and wait for it to come up
echo '::group:: Install CTLog'
make ko-apply-ctlog
echo '::endgroup::'

echo '::group:: Wait for CTLog ready'
kubectl wait --timeout 5m -n ctlog-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n ctlog-system --for=condition=Ready ksvc ctlog
echo '::endgroup::'

# Install TSA and wait for it to come up
echo '::group:: Install TSA'
make ko-apply-tsa
echo '::endgroup::'

echo '::group:: Wait for TSA ready'
kubectl wait --timeout 5m -n tsa-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n tsa-system --for=condition=Ready ksvc tsa
echo '::endgroup::'

# Install tuf
echo '::group:: Install TUF'
make ko-apply-tuf

# Then copy the secrets (even though it's all public stuff, certs, public keys)
# to the tuf-system namespace so that we can construct a tuf root out of it.
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n fulcio-system get secrets fulcio-pub-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n rekor-system get secrets rekor-pub-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n tsa-system get secrets tsa-cert-chain -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
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
FULCIO_GRPC_URL=$(kubectl -n fulcio-system get ksvc fulcio-grpc -ojsonpath='{.status.url}')
export FULCIO_GRPC_URL
CTLOG_URL=$(kubectl -n ctlog-system get ksvc ctlog -ojsonpath='{.status.url}')
export CTLOG_URL
TSA_URL=$(kubectl -n tsa-system get ksvc tsa -ojsonpath='{.status.url}')
export TSA_URL
TUF_MIRROR=$(kubectl -n tuf-system get ksvc tuf -ojsonpath='{.status.url}')
export TUF_MIRROR
