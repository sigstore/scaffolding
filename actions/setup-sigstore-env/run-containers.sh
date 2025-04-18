#!/usr/bin/env bash
#
# Copyright 2025 The Sigstore Authors.
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

set -ex

docker_compose="docker compose"
if ! ${docker_compose} version >/dev/null 2>&1; then
    docker_compose="docker-compose"
fi
echo "setting up OIDC provider"
pushd ./fakeoidc
docker compose up --wait
# the faeoidc container's hostname must be the same, both from within fulcio and from this host machine.
HOST=$(hostname)
OIDC_URL="http://${HOST}:8080"
export OIDC_URL
cat <<EOF > /tmp/fulcio-config.json
{
  "OIDCIssuers": {
    "$OIDC_URL": {
      "IssuerURL": "$OIDC_URL",
      "ClientID": "sigstore",
      "Type": "email"
    }
  }
}
EOF
popd

WORKDIR=$(mktemp -d)
pushd "$WORKDIR"

OIDC_TOKEN="$WORKDIR"/token
export OIDC_TOKEN
curl "$OIDC_URL"/token > "$OIDC_TOKEN"

echo "downloading service repos"
for repo in rekor fulcio timestamp-authority rekor-tiles; do
    if [[ ! -d $repo ]]; then
        git clone https://github.com/sigstore/${repo}.git
    else
        pushd $repo
        git pull
        popd
    fi
done

echo "starting services"
export FULCIO_METRICS_PORT=2113
export FULCIO_CONFIG=/tmp/fulcio-config.json
for repo in rekor fulcio timestamp-authority rekor-tiles; do
    pushd $repo
    # sometimes the services only become healthy after first becoming unhealthy, so we run this command twice.
    ${docker_compose} up --wait || ${docker_compose} up --wait
    if [ "$repo" == "fulcio" ]; then
       docker network inspect fulcio_default | grep fakeoidc || docker network connect --alias "$HOST" fulcio_default fakeoidc
    fi
    popd
done
popd

echo "building trusted root"
./build-trusted-root.sh \
  --fulcio http://localhost:5555 "$WORKDIR/fulcio/config/ctfe/pubkey.pem" \
  --timestamp-url http://localhost:3004 \
  --oidc-url http://localhost:8080 \
  --rekor-v1-url http://localhost:3000 \
  --rekor-v2 http://localhost:3003 "$WORKDIR/rekor-tiles/tests/testdata/pki/ed25519-pub-key.pem"

# set env variables
TSA_URL="http://$(hostname):3004"
CT_LOG_KEY="$WORKDIR/fulcio/config/ctfe/pubkey.pem"
GITHUB_ACTIONS="${GITHUB_ACTIONS:-}"
if [[ -n "$GITHUB_ACTIONS" ]]; then
  # GitHub action env and outputs
  echo "CT_LOG_KEY=$CT_LOG_KEY" >> "$GITHUB_ENV"
  echo "ct-log-key=$CT_LOG_KEY" >> "$GITHUB_OUTPUT"

  echo "OIDC_URL=$OIDC_URL" >> "$GITHUB_ENV"
  echo "oidc-url=$OIDC_URL" >> "$GITHUB_OUTPUT"

  echo "TSA_URL=$TSA_URL" >> "$GITHUB_ENV"
  echo "tsa-url=$TSA_URL" >> "$GITHUB_OUTPUT"

  echo "OIDC_TOKEN=$OIDC_TOKEN" >> "$GITHUB_ENV"
  echo "oidc-token=$OIDC_TOKEN" >> "$GITHUB_OUTPUT"
fi
