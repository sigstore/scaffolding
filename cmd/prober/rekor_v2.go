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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"reflect"
	"time"

	common_v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/rekor-tiles/pkg/client/read"
	"github.com/sigstore/rekor-tiles/pkg/client/write"
	"github.com/sigstore/rekor-tiles/pkg/generated/protobuf"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultRekorV2Origin = "log2025-alpha1.rekor.sigstage.dev"
	rekorV2URL           = "https://" + defaultRekorV2Origin
	readURL              = "https://" + defaultRekorV2Origin + "/api/v2"
)

func prepareRequest(ctx context.Context, privateKey *ecdsa.PrivateKey) (*protobuf.HashedRekordRequestV002, error) {
	artifact := []byte(time.Now().String())
	digest := sha256.Sum256(artifact)
	sig, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		return nil, err
	}
	publicKey, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return nil, err
	}

	request := &protobuf.HashedRekordRequestV002{
		Signature: &protobuf.Signature{
			Content: sig,
			Verifier: &protobuf.Verifier{
				Verifier: &protobuf.Verifier_PublicKey{
					PublicKey: &protobuf.PublicKey{
						RawBytes: publicKey,
					},
				},
				KeyDetails: common_v1.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256,
			},
		},
		Digest: digest[:],
	}
	return request, nil
}

func retrieveEntry(ctx context.Context, privateKey *ecdsa.PrivateKey, logIndex uint64, treeSize uint64) (*protobuf.Entry, error) {
	tileIndex := treeSize / layout.TileWidth
	partial := layout.PartialTileSize(0, logIndex, treeSize)

	verifier, err := signature.LoadDefaultSignerVerifier(privateKey)
	if err != nil {
		return nil, err
	}
	readClient, err := read.NewReader(readURL, defaultRekorV2Origin, verifier)
	if err != nil {
		return nil, err
	}
	entryBundleBytes, err := readClient.ReadEntryBundle(ctx, tileIndex, partial)
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
	// after the fethcing the bubdle at the particular partial, our target entry will always be the last.
	err = protojson.Unmarshal(entryBundle.Entries[len(entryBundle.Entries)-1], readEntry)
	if err != nil {
		return nil, err
	}
	return readEntry, nil
}

func TestAddAndRetrieveEntry(ctx context.Context) error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		Logger.Fatalf("failed to generate key: %v", err)
	}
	request, err := prepareRequest(ctx, privateKey)
	if err != nil {
		return err
	}

	// submit the entry
	writeClient, err := write.NewWriter(rekorV2URL)
	if err != nil {
		return err
	}
	tLE, err := writeClient.Add(ctx, request)
	if err != nil {
		return err
	}

	// submit additional unused requests
	request2, err := prepareRequest(ctx, privateKey)
	if err != nil {
		return err
	}
	if _, err = writeClient.Add(ctx, request2); err != nil {
		return err
	}
	request3, err := prepareRequest(ctx, privateKey)
	if err != nil {
		return err
	}
	if _, err = writeClient.Add(ctx, request3); err != nil {
		return err
	}

	// parse the submitted entry
	writtenEntry := &protobuf.Entry{}
	err = protojson.Unmarshal(tLE.CanonicalizedBody, writtenEntry)
	if err != nil {
		return err
	}
	logIndex := uint64(tLE.InclusionProof.LogIndex)
	treeSize := uint64(tLE.InclusionProof.TreeSize)

	// retrieve the entry
	readEntry, err := retrieveEntry(ctx, privateKey, logIndex, treeSize)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(writtenEntry, readEntry) {
		return errors.New("submitted and retrieved entries do not match")
	}
	return nil
}
