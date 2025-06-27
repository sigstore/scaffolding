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


# <cmd> || return is so the script can exit early without quitting your shell.

CLONE_DIR="${CLONE_DIR:-$(mktemp -d)}"
CWD="$(pwd)"

echo "setting up OIDC provider"
pushd ./fakeoidc || return
docker compose up --wait
# the fakeoidc container's hostname must be the same, both from within fulcio and from this host machine.
HOST="${HOST:-$(hostname)}"
export OIDC_URL="http://${HOST}:8080"
export FULCIO_CONFIG=$CLONE_DIR/fulcio-config.json
cat <<EOF > "$FULCIO_CONFIG"
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
popd || return

echo "downloading service repos"
pushd "$CLONE_DIR" || return
FULCIO_REPO="${FULCIO_REPO:-sigstore/fulcio}"
REKOR_REPO="${REKOR_REPO:-sigstore/rekor}"
TIMESTAMP_AUTHORITY_REPO="${TIMESTAMP_AUTHORITY_REPO:-sigstore/timestamp-authority}"
REKOR_TILES_REPO="${REKOR_TILES_REPO:-sigstore/rekor-tiles}"
OWNER_REPOS=(
  "$FULCIO_REPO"
  "$REKOR_REPO"
  "$TIMESTAMP_AUTHORITY_REPO"
  "$REKOR_TILES_REPO"
)
procs=${#OWNER_REPOS[@]}
for owner_repo in "${OWNER_REPOS[@]}"; do
    repo=$(basename "$owner_repo")
    if [[ ! -d $repo ]]; then
        echo "'git clone https://github.com/${owner_repo}.git'"
    else
        echo "'cd $repo && git pull'"
    fi
done | xargs -P "$procs" -l1 bash -c
export CT_LOG_KEY="$CLONE_DIR/fulcio/config/ctfe/pubkey.pem"

echo "starting services"
export FULCIO_METRICS_PORT=2113
for owner_repo in "${OWNER_REPOS[@]}"; do
    repo=$(basename "$owner_repo")
    echo "'cd $repo && docker compose up --wait'"
done | xargs -P "$procs" -l1 bash -c
docker network inspect fulcio_default | grep fakeoidc || docker network connect --alias "$HOST" fulcio_default fakeoidc || return
export TSA_URL="http://${HOST}:3004"
popd || return

export OIDC_TOKEN="$CLONE_DIR"/token
curl -o "$OIDC_TOKEN" "$OIDC_URL"/token || return

stop_services() {
  pushd ./fakeoidc || return
  docker compose down --volumes
  popd || return
  pushd "$CLONE_DIR" || return
  for owner_repo in "${OWNER_REPOS[@]}"; do
    repo=$(basename "$owner_repo")
    pushd "$repo" || return
    docker compose down --volumes
    popd || return
  done
  popd || return
}

echo "building trusted root"
pushd "$CLONE_DIR" || return
"$CWD"/build-trusted-root.sh \
  --fulcio http://localhost:5555 "$CLONE_DIR/fulcio/config/ctfe/pubkey.pem" \
  --timestamp-url http://localhost:3004 \
  --oidc-url http://localhost:8080 \
  --rekor-v1-url http://localhost:3000 \
  --rekor-v2 http://localhost:3003 "$CLONE_DIR/rekor-tiles/tests/testdata/pki/ed25519-pub-key.pem" "rekor-local" \
  || return
export TRUSTED_ROOT="$CLONE_DIR/trusted_root.json"
export SIGNING_CONFIG="$CLONE_DIR/signing_config.json"
export TRUST_CONFIG="$CLONE_DIR/trust_config.json"
popd || return
