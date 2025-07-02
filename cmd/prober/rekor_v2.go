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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	common_v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/rekor-tiles/pkg/generated/protobuf"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
	"google.golang.org/protobuf/encoding/protojson"
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

func retrieveRekorV2Entry(rekorV2URL string, logIndex, treeSize uint64) (*protobuf.Entry, error) {
	entriesPath := layout.EntriesPathForLogIndex(logIndex, treeSize)
	entryBundleBytes, err := observeRequest(rekorV2URL, ReadProberCheck{
		Endpoint: fmt.Sprintf("/api/v2/%s", entriesPath),
		Method:   GET,
	})
	if err != nil {
		return nil, err
	}
	entryBundle := api.EntryBundle{}
	err = entryBundle.UnmarshalText(entryBundleBytes)
	if err != nil {
		return nil, err
	}
	readEntry := &protobuf.Entry{}
	// TODO: confirm
	// after the fetching the bubdle at the particular partial, our target entry will always be the last.
	err = protojson.Unmarshal(entryBundle.Entries[len(entryBundle.Entries)-1], readEntry)
	if err != nil {
		return nil, err
	}
	return readEntry, nil
}

// rekorV2WriteEndpoint creates and sends an entry to the rekorV2 instance.
func rekorV2WriteEndpoint(ctx context.Context, cert *x509.Certificate, priv *ecdsa.PrivateKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	artifact := []byte(time.Now().String())
	digest := sha256.Sum256(artifact)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		return err
	}
	verifier := &protobuf.Verifier{
		KeyDetails: common_v1.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256,
	}
	if cert != nil {
		verifier.Verifier = &protobuf.Verifier_X509Certificate{
			X509Certificate: &common_v1.X509Certificate{
				RawBytes: cert.Raw,
			},
		}
	} else {
		pubBytes, err := cryptoutils.MarshalPublicKeyToDER(priv.Public())
		if err != nil {
			return err
		}
		verifier.Verifier = &protobuf.Verifier_PublicKey{
			PublicKey: &protobuf.PublicKey{
				RawBytes: pubBytes,
			},
		}
	}
	createEntryRequest := &protobuf.CreateEntryRequest{
		Spec: &protobuf.CreateEntryRequest_HashedRekordRequestV002{
			HashedRekordRequestV002: &protobuf.HashedRekordRequestV002{
				Signature: &protobuf.Signature{
					Content:  sig,
					Verifier: verifier,
				},
				Digest: digest[:],
			},
		},
	}
	reqBytes, err := protojson.Marshal(createEntryRequest)
	if err != nil {
		return err
	}
	proberCheck := ReadProberCheck{
		Endpoint: "/api/v2/log/entries",
		Method:   POST,
		Body:     reqBytes,
	}
	respBytes, err := observeRequest(rekorV2URL, proberCheck)
	if err != nil {
		return err
	}
	tle := v1.TransparencyLogEntry{}
	if err := protojson.Unmarshal(respBytes, &tle); err != nil {
		return err
	}
	tleBody := protobuf.Entry{}
	if err := protojson.Unmarshal(tle.CanonicalizedBody, &tleBody); err != nil {
		return err
	}
	// basic content matching, not necessarily any signature verification.
	tleBodyDigest := tleBody.Spec.GetHashedRekordV002().Data.Digest
	if !bytes.Equal(tleBodyDigest, digest[:]) {
		return fmt.Errorf("tleEntry digest does not match: got: %s, want: %s", tleBodyDigest, digest)
	}
	tleEntrySig := tleBody.Spec.GetHashedRekordV002().Signature.Content
	if !bytes.Equal(tleEntrySig, sig) {
		return fmt.Errorf("tleEntry signature does not match: got: %s, want: %s", tleEntrySig, sig)
	}
	retrievedEntry, err := retrieveRekorV2Entry(rekorV2URL, uint64(tle.InclusionProof.LogIndex), uint64(tle.InclusionProof.TreeSize))
	if err != nil {
		return err
	}
	// basic content matching, not necessarily any signature verification.
	receivedDigest := retrievedEntry.Spec.GetHashedRekordV002().Data.Digest
	if !bytes.Equal(receivedDigest, digest[:]) {
		return fmt.Errorf("retrieved entry digests do not match: got: %s, want: %s", receivedDigest, digest)
	}
	receivedSig := retrievedEntry.Spec.GetHashedRekordV002().Signature.Content
	if !bytes.Equal(receivedSig, sig) {
		return fmt.Errorf("retrieved entry signatures do not match: got: %s, want: %s", receivedSig, sig)
	}
	return nil
}
