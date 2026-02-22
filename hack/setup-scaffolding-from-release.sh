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
set -o xtrace

# Default
RELEASE_VERSION="v0.7.24"

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
TSA=https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-tsa.yaml

# Since things that we install vary based on the release version, parse out
# MAJOR, MINOR, and PATCH
# We don't use MAJOR yet, but add it here for future.
# MAJOR=$(echo "$RELEASE_VERSION" | cut -d '.' -f 1 | sed -e 's/v//')
MINOR=$(echo "${RELEASE_VERSION}" | cut -d '.' -f 2)
PATCH=$(echo "${RELEASE_VERSION}" | cut -d '.' -f 3)

if [[ "${MINOR}" -lt 4 ]]; then
  echo Unsupported version, only support versions >= 0.4.0
  exit 1
fi

# We introduced TSA in release v0.5.0
INSTALL_TSA="false"
if [[ "${MINOR}" -ge 5 ]]; then
  INSTALL_TSA="true"
fi

# Since the behaviour on oidc is different on certain k8s versions, check to see if we
# need to do some mucking with the Fulcio config
NEED_TO_UPDATE_FULCIO_CONFIG="false"
K8S_SERVER_VERSION=$(kubectl version -ojson | yq '.serverVersion.minor' -)

if [[ "${K8S_SERVER_VERSION}" == "21" ]] || [[ "${K8S_SERVER_VERSION}" == "22" ]]; then
  echo "Running on k8s 1.${K8S_SERVER_VERSION}.x will update Fulcio accordingly"
  NEED_TO_UPDATE_FULCIO_CONFIG="true"
fi

cleanup_cmd=""
cleanup() {
    eval "$cleanup_cmd"
}
trap cleanup EXIT

# Install Trillian if it is part of this release and wait for it to come up
echo '::group:: Install Trillian'
has_trillian=1
if curl -s -i https://github.com/sigstore/scaffolding/releases/download/${RELEASE_VERSION}/release-trillian.yaml | grep 'HTTP/2 404' ; then
  has_trillian=0
fi
[ -z $has_trillian ] || kubectl apply -f "${TRILLIAN}"
echo '::endgroup::'

echo '::group:: Wait for Trillian ready'
[ -z $has_trillian ] || kubectl wait --timeout 5m -n trillian-system --for=condition=Ready ksvc log-server
[ -z $has_trillian ] || kubectl wait --timeout 5m -n trillian-system --for=condition=Ready ksvc log-signer
echo '::endgroup::'

# Install Rekor and wait for it to come up
echo '::group:: Install Rekor'

pass=$(uuidgen)
rekordir=$(mktemp -d)
openssl genpkey -algorithm ed25519 -out "${rekordir}/key.pem" -pass pass:"${pass}"
openssl pkey -in "${rekordir}/key.pem" -out "${rekordir}/pub.pem" -pubout
cleanup_rekor() {
    rm "${rekordir}/pub.pem" "${rekordir}/key.pem"
}
cleanup_cmd="cleanup_rekor"
kubectl apply -f "${REKOR}"
curl -Ls "${REKOR}" | sed -e "s/<private-placeholder>/$(cat "${rekordir}/key.pem" | base64 -w0)/" \
  -e "s/<public-placeholder>/$(cat "${rekordir}/pub.pem" | base64 -w0)/" \
  -e "s/<password-placeholder>/$(echo -n "$pass" | base64 -w0)/" | \
  kubectl apply -f -
echo '::endgroup::'

echo '::group:: Wait for Rekor ready'
kubectl wait --timeout 5m -n rekor-system --for=condition=Complete jobs --all
kubectl wait --timeout 5m -n rekor-system --for=condition=Ready ksvc rekor
echo '::endgroup::'

# Install Fulcio and wait for it to come up
echo '::group:: Install Fulcio'
fulcio=$(mktemp --tmpdir fulcioXXX)
curl -Ls -o "${fulcio}" "${FULCIO}"
if [[ "${NEED_TO_UPDATE_FULCIO_CONFIG}" == "true" ]]; then
  echo "Fixing Fulcio config for < 1.23.X Kubernetes"
  sed -i -e 's@https://kubernetes.default.svc.cluster.local@https://kubernetes.default.svc@' "${fulcio}"
fi
pass=$(uuidgen)
fulciodir=$(mktemp -d)
openssl ecparam -name prime256v1 -genkey | openssl pkcs8 -passout "pass:${pass}" -topk8 -out "${fulciodir}/key.pem"
openssl req -x509 -new -key "${fulciodir}/key.pem" -out "${fulciodir}/cert.pem" -sha256 -days 10 -subj "/O=test/CN=fulcio.scaffolding.test" -passin "pass:${pass}"
cleanup_fulcio() {
    rm "${fulciodir}/cert.pem" "${fulciodir}/key.pem"
}
cleanup_cmd="$cleanup_cmd ; cleanup_fulcio"
sed -i -e "s/<private-placeholder>/$(cat "${fulciodir}/key.pem" | base64 -w0)/" \
  -e "s/<cert-placeholder>/$(cat "${fulciodir}/cert.pem" | base64 -w0)/" \
  -e "s/<password-placeholder>/$(echo -n "$pass" | base64 -w0)/" "${fulcio}"
kubectl apply -f "${fulcio}"
rm "${fulcio}"

kubectl get -n fulcio-system cm fulcio-config -o json

echo '::group:: Wait for Fulcio ready'
kubectl -n fulcio-system get job 2>&1 | grep 'No resources found' || kubectl wait --timeout 5m -n fulcio-system --for=condition=Complete jobs --all
kubectl wait --timeout 5m -n fulcio-system --for=condition=Ready ksvc fulcio
# this checks if the requested version is > 0.4.12 (and therefore has fulcio-grpc in it)
if [[ "${PATCH}" -gt 12 ]] || [[ "${MINOR}" -ge 5 ]]; then
  kubectl wait --timeout 5m -n fulcio-system --for=condition=Ready ksvc fulcio-grpc
fi
echo '::endgroup::'

# Install CTlog and wait for it to come up
echo '::group:: Install CTLog'
ctdir=$(mktemp -d)
openssl ecparam -name prime256v1 -genkey -noout -out "${ctdir}/key.pem"
openssl ec -in "${ctdir}/key.pem" -pubout -out "${ctdir}/pub.pem"
cleanup_ctlog() {
    rm "${ctdir}/pub.pem" "${ctdir}/key.pem"
}
cleanup_cmd="$cleanup_cmd ; cleanup_ctlog"
curl -Ls "${CTLOG}" | sed -e "s/<private-placeholder>/$(cat "${ctdir}/key.pem" | base64 -w0)/" \
  -e "s/<public-placeholder>/$(cat "${ctdir}/pub.pem" | base64 -w0)/" \
  -e "s/<cert-placeholder>/$(cat "${fulciodir}/cert.pem" | base64 -w0)/" | \
  kubectl apply -f -
echo '::endgroup::'

echo '::group:: Wait for CTLog ready'
kubectl wait --timeout 5m -n ctlog-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n ctlog-system --for=condition=Ready ksvc ctlog
echo '::endgroup::'

# If we're running release > 0.5.0 install TSA
if [[ "${INSTALL_TSA}" == "true" ]]; then
kubectl apply -f "${TSA}"
kubectl wait --timeout 5m -n tsa-system --for=condition=Complete jobs --all
kubectl wait --timeout 2m -n tsa-system --for=condition=Ready ksvc tsa
fi

# Install tuf
echo '::group:: Install TUF'
kubectl apply -f "${TUF}"

# Then copy the secrets (even though it's all public stuff, certs, public keys)
# to the tuf-system namespace so that we can construct a tuf root out of it.
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed -e '/creationTimestamp:/d' -e '/uid:/d' -e '/resourceVersion:/d' -e 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n fulcio-system get secrets fulcio-pub-key -oyaml | sed -e '/creationTimestamp:/d' -e '/uid:/d' -e '/resourceVersion:/d' -e 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n rekor-system get secrets rekor-pub-key -oyaml | sed -e '/creationTimestamp:/d' -e '/uid:/d' -e '/resourceVersion:/d' -e 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -

if [[ "${INSTALL_TSA}" == "true" ]]; then
kubectl -n tsa-system get secrets tsa-cert-chain -oyaml | sed -e '/creationTimestamp:/d' -e '/uid:/d' -e '/resourceVersion:/d' -e 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
fi
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

if [[ "${INSTALL_TSA}" == "true" ]]; then
  TSA_URL=$(kubectl -n tsa-system get ksvc tsa -ojsonpath='{.status.url}')
  export TSA_URL
fi
