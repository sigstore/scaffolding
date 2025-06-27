// Copyright 2025 The Sigstore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	common_v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/rekor-tiles/pkg/generated/protobuf"
	"github.com/transparency-dev/tessera/api/layout"
)

// getTreeSize returns the size of the rekorV2 log tree.
func getTreeSize(rekorV2URL string) (int, error) {
	req, err := retryablehttp.NewRequest("GET", rekorV2URL+"/api/v2/checkpoint", nil)
	if err != nil {
		return 0, fmt.Errorf("invalid request for checkpoint: %w", err)
	}

	setHeaders(req, "", ReadProberCheck{})
	resp, err := retryableClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("unexpected error getting loginfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected response code received from loginfo endpoint: %w", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading loginfo body: %w", err)
	}

	// The second line will be the log size.
	// See https://github.com/C2SP/C2SP/blob/94c93bee35b922c91b3729c7f184ce3104a6c7cb/tlog-checkpoint.md#note-text.
	logSizeBytes := bytes.Split(bodyBytes, []byte{'\n'})[1]
	logSize, err := strconv.Atoi(string(logSizeBytes))
	if err != nil {
		return 0, fmt.Errorf("parsing log size: %w", err)
	}
	return logSize, nil
}

// determineRekorV2ShardCoverage determines which endpoints to check for a given rekorV2 shard host.
// See https://github.com/sigstore/rekor-tiles/blob/98cd4a77300f81eb79ca50f5b8d70ee2a00cbd50/api/proto/rekor/v2/rekor_service.proto#L74.
func determineRekorV2ShardCoverage(rekorV2URL string) ([]*ReadProberCheck, error) {
	treeSize, err := getTreeSize(rekorV2URL)
	if err != nil {
		return nil, err
	}
	tileIndex := treeSize / layout.TileWidth
	level := 0 // We always want the items at the botton of the tree (leaf nodes).
	partial := layout.PartialTileSize(uint64(level), uint64(tileIndex), uint64(treeSize))
	tilePath := layout.TilePath(uint64(level), uint64(tileIndex), partial)
	entriesPath := layout.EntriesPath(uint64(tileIndex), partial)
	proberChecks := []*ReadProberCheck{
		{
			Endpoint: "/api/v2/checkpoint",
			Method:   GET,
		},
		{
			Endpoint: fmt.Sprintf("/api/v2/%s", tilePath),
			Method:   GET,
		},
		{
			Endpoint: fmt.Sprintf("/api/v2/%s", entriesPath),
			Method:   GET,
		},
	}
	return proberChecks, nil
}

// rekorV2WriteEndpoint creates and sends an entry to the rekorV2 instance.
func rekorV2WriteEndpoint(ctx context.Context, cert *x509.Certificate, priv *ecdsa.PrivateKey) error {
	artifact := []byte(time.Now().String())
	digest := sha256.Sum256(artifact)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		return err
	}

	createEntryRequest := &protobuf.CreateEntryRequest{
		Spec: &protobuf.CreateEntryRequest_HashedRekordRequestV002{
			HashedRekordRequestV002: &protobuf.HashedRekordRequestV002{
				Signature: &protobuf.Signature{
					Content: sig,
					Verifier: &protobuf.Verifier{
						Verifier: &protobuf.Verifier_X509Certificate{
							X509Certificate: &common_v1.X509Certificate{
								RawBytes: cert.Raw,
							},
						},
						KeyDetails: common_v1.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256,
					},
				},
				Digest: digest[:],
			},
		},
	}
	reqBytes, err := json.Marshal(createEntryRequest)
	if err != nil {
		return err
	}
	proberCheck := ReadProberCheck{
		Endpoint: "/api/v2/log/entries",
		Method:   POST,
		Body:     reqBytes,
	}
	if err := observeRequest(rekorV2URL, proberCheck); err != nil {
		return err
	}
	return nil
}
