#!/bin/bash
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


# Build a Sigstore trustedroot.json and signingconfig.json using a running test instance and source dirs
#
# Requires cosign binary to be in PATH.
# Options can be given multiple times (signing config will contain only the last URLs,
# but trustedroot will contain every instance):
#  --fulcio <INSTANCEURL> <KEYFILE>
#  --rekor-v1-url <INSTANCEURL>
#  --rekor-v2 <INSTANCEURL> <KEYFILE>
#  --timestamp-url <INSTANCEURL>

set -euo pipefail


[ -f trusted_root.json -o -f signing_config.json ] && echo "trusted_root.json or signing_config.json already exist" && exit 1 

CMD="cosign trusted-root create"
WORKDIR=$(mktemp -d)

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --fulcio)
            FULCIO_URL="$2"
            KEYFILE="$3"
            shift
            shift

            FNAME=$(mktemp --tmpdir="$WORKDIR" fulcio_cert.XXXX.pem)
            curl --fail -o "$FNAME" "$FULCIO_URL"/api/v1/rootCert
            CMD="$CMD --certificate-chain $FNAME"

            CMD="$CMD --ctfe-key $KEYFILE"            
            ;;

        --rekor-v1-url)
            REKOR_URL="$2"
            shift

            FNAME=$(mktemp --tmpdir="$WORKDIR" rekorv1_pub.XXXX.pem)
            curl --fail -o "$FNAME" "$REKOR_URL"/api/v1/log/publicKey
            CMD="$CMD --rekor-key $FNAME"
            ;;

        --rekor-v2)
            REKOR_URL="$2"
            KEYFILE="$3"
            shift
            shift

            CMD="$CMD --rekor-key $KEYFILE"
            ;;

        --timestamp-url)
            URL="$2"
            shift

            FNAME=$(mktemp --tmpdir="$WORKDIR" timestamp_certs.XXXX.pem)
            curl --fail -o "$FNAME" "$URL"/api/v1/timestamp/certchain
            CMD="$CMD --timestamp-certificate-chain $FNAME"
            ;;

        --oidc-url)
            OIDC_URL="$2"
            shift
            ;;

        *) echo "Unknown parameter passed: $1"; 
            exit 1
            ;;
    esac
    shift
done

$CMD > trusted_root.json

# construct a signingconfig as well
cat << EOF > signing_config.json
{
  "mediaType": "application/vnd.dev.sigstore.signingconfig.v0.1+json",
  "caUrl": "$FULCIO_URL",
  "oidcUrl": "$OIDC_URL",
  "tlogUrls": [
    "$REKOR_URL"
  ]
}
EOF

echo "Wrote trusted_root.json & signing_config.json"

CWD="$(pwd)"
export TRUSTED_ROOT="$CWD/trusted_root.json"
export SIGNING_CONFIG="$CWD/signing_config.json"
if [[ -n "$GITHUB_ACTIONS" ]]; then
  # GitHub action env and outputs
  echo "TRUSTED_ROOT=$(echo "$TRUSTED_ROOT")" >> "$GITHUB_ENV"
  echo "trusted-root=$(echo "$TRUSTED_ROOT")" >> "$GITHUB_OUTPUT"

  echo "SIGNING_CONFIG=$(echo "$SIGNING_CONFIG")" >> "$GITHUB_ENV"
  echo "signing-config=$(echo "$SIGNING_CONFIG")" >> "$GITHUB_OUTPUT"
fi
