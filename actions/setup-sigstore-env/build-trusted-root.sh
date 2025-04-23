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

WORKDIR=$(mktemp -d)
# run cosign as a container with the current user permissions. This script will copy files into $WORKDIR.
COSIGN_CMD="docker run --user=$(id -u):$(id -g) --rm -v $WORKDIR/:$WORKDIR/ ghcr.io/sigstore/cosign/cosign:latest"
CMD="$COSIGN_CMD trusted-root create"

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --fulcio)
            FULCIO_URL="$2"
            KEYFILE="$3"
            shift
            shift

            cp "$KEYFILE" "$WORKDIR"/
            KEYFILE=$WORKDIR/$(basename "$KEYFILE")

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

            cp "$KEYFILE" "$WORKDIR"/
            KEYFILE=$WORKDIR/$(basename "$KEYFILE")

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
