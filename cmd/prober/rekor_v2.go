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
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	"github.com/sigstore/rekor-tiles/pkg/client/write"
	"github.com/sigstore/rekor-tiles/pkg/generated/protobuf"
	"github.com/transparency-dev/tessera/api/layout"
)

const (
	defaultRekorV2URL = "https://log2025-alpha1.rekor.sigstage.dev"
)

func AddRekorV2Entry(ctx context.Context) error {
	artifact := []byte(time.Now().String())
	digest := sha256.Sum256(artifact)

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		Logger.Fatalf("failed to generate key: %v", err)
	}
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		return err
	}
	publicKey, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return err
	}

	request := &protobuf.HashedRekordRequestV002{
		Signature: &protobuf.Signature{
			Content: signature,
			Verifier: &protobuf.Verifier{
				Verifier: &protobuf.Verifier_PublicKey{
					PublicKey: &protobuf.PublicKey{
						RawBytes: publicKey,
					},
				},
				KeyDetails: v1.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256,
			},
		},
		Digest: digest[:],
	}

	writeClient, err := write.NewWriter(defaultRekorV2URL)
	if err != nil {
		return err
	}
	entry, err := writeClient.Add(ctx, request)
	if err != nil {
		return err
	}
	spew.Dump(entry)

	logIndex := entry.InclusionProof.LogIndex
	treeSize := entry.InclusionProof.TreeSize
	// tileIndex := logIndex / layout.EntryBundleWidth
	// partial := logIndex % layout.EntryBundleWidth

	path := layout.EntriesPathForLogIndex(uint64(logIndex), uint64(treeSize))
	print(fmt.Sprintf("\n %s", path))
	_, partial, err := layout.ParseTileIndexPartial(path)
	if err != nil {
		return err
	}
	print(fmt.Sprintf("\n %d", partial))

	return nil
}
