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
# Options can be given multiple times:
#  --fulcio <INSTANCEURL> <KEYFILE>
#  --rekor-v1-url <INSTANCEURL>
#  --rekor-v2 <INSTANCEURL> <KEYFILE>
#  --timestamp-url <INSTANCEURL>

set -euo pipefail

WORKDIR=$(mktemp -d)
# run cosign as a container with the current user permissions. This script will copy files into $WORKDIR.
SCRIPT_DIR=$(dirname "$(realpath "$0")")
docker build ./ -f "$SCRIPT_DIR"/Dockerfile.cosign -t cosign
COSIGN_CMD="docker run --user=$(id -u):$(id -g) --rm -v $WORKDIR/:$WORKDIR/ cosign"
CMD="$COSIGN_CMD trusted-root create"

FULCIO_SIGNING_CONFIGS=""

add_fulcio_to_signing_config () {
  if [ -n "$FULCIO_SIGNING_CONFIGS" ]; then
    FULCIO_SIGNING_CONFIGS="$FULCIO_SIGNING_CONFIGS,
    "
  fi
  FULCIO_SIGNING_CONFIGS="$FULCIO_SIGNING_CONFIGS{
      \"url\": \"$1\",
      \"majorApiVersion\": 1,
      \"validFor\": { \"start\": \"2025-05-25T00:00:00Z\" },
      \"operator\": \"scaffolding-setup-sigstore-env\"
    }"
}

REKOR_SIGNING_CONFIGS=""

add_rekor_to_signing_config () {
  if [ -n "$REKOR_SIGNING_CONFIGS" ]; then
    REKOR_SIGNING_CONFIGS="$REKOR_SIGNING_CONFIGS,
    "
  fi
  REKOR_SIGNING_CONFIGS="$REKOR_SIGNING_CONFIGS{
      \"url\": \"$1\",
      \"majorApiVersion\": $2,
      \"validFor\": { \"start\": \"2025-05-25T00:00:00Z\" },
      \"operator\": \"scaffolding-setup-sigstore-env\"
    }"
}

TSA_SIGNING_CONFIGS=""

add_tsa_to_signing_config () {
  if [ -n "$TSA_SIGNING_CONFIGS" ]; then
    TSA_SIGNING_CONFIGS="$TSA_SIGNING_CONFIGS,
    "
  fi
  TSA_SIGNING_CONFIGS="$TSA_SIGNING_CONFIGS{
      \"url\": \"$1/api/v1/timestamp\",
      \"majorApiVersion\": 1,
      \"validFor\": { \"start\": \"2025-05-25T00:00:00Z\" },
      \"operator\": \"scaffolding-setup-sigstore-env\"
    }"
}

while [[ "$#" -gt 0 ]]; do
  case $1 in
    --fulcio)
      FULCIO_URL="$2"
      KEYFILE="$3"
      shift
      shift

      add_fulcio_to_signing_config "$FULCIO_URL"

      # copy to our WORKDIR to be mounted in our cosign container.
      cp "$KEYFILE" "$WORKDIR"/
      KEYFILE=$WORKDIR/$(basename "$KEYFILE")

      FNAME=$(mktemp --tmpdir="$WORKDIR" fulcio_cert.XXXX.pem)
      curl --fail -o "$FNAME" "$FULCIO_URL"/api/v1/rootCert
      CMD="$CMD --certificate-chain $FNAME --fulcio-uri $FULCIO_URL"

      CMD="$CMD --ctfe-key $KEYFILE"
      ;;

    --rekor-v1-url)
      URL="$2"
      shift

      add_rekor_to_signing_config "$URL" 1

      FNAME=$(mktemp --tmpdir="$WORKDIR" rekorv1_pub.XXXX.pem)
      curl --fail -o "$FNAME" "$URL"/api/v1/log/publicKey
      CMD="$CMD --rekor-key $FNAME --rekor-url $URL"
      ;;

    --rekor-v2)
      URL="$2"
      KEYFILE="$3"
      HOST="$4"
      shift
      shift
      shift

      add_rekor_to_signing_config "$URL" 2

      # copy to our WORKDIR to be mounted in our cosign container.
      cp "$KEYFILE" "$WORKDIR"/
      KEYFILE=$WORKDIR/$(basename "$KEYFILE")

      CMD="$CMD --rekor-key $KEYFILE,$HOST --rekor-url http://$HOST"
      ;;

    --timestamp-url)
      URL="$2"
      shift

      add_tsa_to_signing_config "$URL"

      FNAME=$(mktemp --tmpdir="$WORKDIR" timestamp_certs.XXXX.pem)
      curl --fail -o "$FNAME" "$URL"/api/v1/timestamp/certchain
      CMD="$CMD --timestamp-certificate-chain $FNAME --timestamp-uri $URL"
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
  "mediaType": "application/vnd.dev.sigstore.signingconfig.v0.2+json",
  "caUrls": [
    $FULCIO_SIGNING_CONFIGS
  ],
  "oidcUrls": [
    {
      "url": "$OIDC_URL",
      "majorApiVersion": 1,
      "validFor": { "start": "2025-05-25T00:00:00Z" },
      "operator": "scaffolding-setup-sigstore-env"
    }
  ],
  "rekorTlogUrls": [
    $REKOR_SIGNING_CONFIGS
  ],
  "tsaUrls": [
    $TSA_SIGNING_CONFIGS
  ],
  "rekorTlogConfig": { "selector": "ANY" },
  "tsaConfig": { "selector": "ANY" }
}
EOF

# finally build a trustconfig (trustedroot + signingconfig)
cat << EOF >trust_config.json
{
"mediaType": "application/vnd.dev.sigstore.clienttrustconfig.v0.1+json",
"trustedRoot": $(cat trusted_root.json),
"signingConfig": $(cat signing_config.json)
}
EOF



echo "Wrote trusted_root.json, signing_config.json & trust_config.json"
