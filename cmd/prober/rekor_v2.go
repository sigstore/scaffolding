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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/digitorus/timestamp"
	common_v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/rekor-tiles/pkg/client/read"
	"github.com/sigstore/rekor-tiles/pkg/client/write"
	"github.com/sigstore/rekor-tiles/pkg/generated/protobuf"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	tsa_client "github.com/sigstore/timestamp-authority/pkg/client"
	tsa_verification "github.com/sigstore/timestamp-authority/pkg/verification"

	ts "github.com/sigstore/timestamp-authority/pkg/generated/client/timestamp"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	defaultRekorV2Origin = "log2025-alpha1.rekor.sigstage.dev"
	rekorV2URL           = "https://" + defaultRekorV2Origin
	readURL              = "https://" + defaultRekorV2Origin + "/api/v2"
)

// timestamp fetches a timestamp and verifies it upon retrieval.
func submitAndVerifyTimestamp(ctx context.Context, signature []byte) error {
	signatureHash := sha256.Sum256(signature)

	client, err := tsa_client.GetTimestampClient(tsaURL)
	if err != nil {
		return fmt.Errorf("creating the timestamp client: %w", err)
	}

	getTSReq := &timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: signatureHash[:],
	}
	getTSReqBytes, err := getTSReq.Marshal()
	if err != nil {
		return fmt.Errorf("marshalling the timestamp request: %w", err)
	}
	var getTSRespBytes bytes.Buffer
	_, err = client.Timestamp.GetTimestampResponse(
		ts.NewGetTimestampResponseParams().WithContext(ctx).WithRequest(io.NopCloser(bytes.NewReader(getTSReqBytes))),
		&getTSRespBytes,
	)
	if err != nil {
		return fmt.Errorf("getting the timestamp response: %w", err)
	}
	Logger.Debug("submitted timestamp")

	getCertResp, err := client.Timestamp.GetTimestampCertChain(ts.NewGetTimestampCertChainParamsWithContext(ctx))
	if err != nil {
		return fmt.Errorf("getting the timestamp cert chain: %w", err)
	}
	certs, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(getCertResp.Payload))
	if err != nil {
		return fmt.Errorf("parsing the cert chain: %w", err)
	}
	_, err = tsa_verification.VerifyTimestampResponse(getTSRespBytes.Bytes(), bytes.NewReader(signature), tsa_verification.VerifyOpts{
		TSACertificate: certs[0],                // Explicitly provide the leaf certificate.
		Intermediates:  certs[1 : len(certs)-1], // Intermediates start from the second cert.
		Roots:          certs[len(certs)-1:],    // Root is the last cert.
	})
	if err != nil {
		return fmt.Errorf("verifying the timestamp: %w", err)
	}
	Logger.Debug("verified timestamp")
	return nil
}

func retrieveRekorV2Entry(ctx context.Context, privateKey *ecdsa.PrivateKey, logIndex, treeSize uint64) (*protobuf.Entry, error) {
	tileIndex := treeSize / layout.TileWidth
	level := uint(0) // We always want the items at the botton of the tree (leaf nodes).
	partial := layout.PartialTileSize(uint64(level), logIndex, treeSize)

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
	Logger.Debug("retrieved Rekor V2 entry")
	return readEntry, nil
}

// submitRekorV2Entry sends an entry to Rekor V2.
func submitRekorV2Entry(ctx context.Context, digest []byte, sig []byte, cert *x509.Certificate) (*v1.TransparencyLogEntry, error) {
	request := &protobuf.HashedRekordRequestV002{
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
	}
	writeClient, err := write.NewWriter(rekorV2URL)
	if err != nil {
		return nil, err
	}
	tLE, err := writeClient.Add(ctx, request)
	if err != nil {
		return nil, err
	}
	Logger.Debug("submitted Rekor V2 entry")
	return tLE, nil
}

// submitAndRetrieveRekorV2Entry sends an antry to Rekor V2 and also retreieves it.
func submitAndRetrieveRekorV2Entry(ctx context.Context, privateKey *ecdsa.PrivateKey, digest []byte, sig []byte, cert *x509.Certificate) error {
	tLE, err := submitRekorV2Entry(ctx, digest, sig, cert)
	if err != nil {
		return err
	}

	// submit a few more entries, to make sure can retrieve our target entry, despite any offset from the tail fo the log.
	for range 3 {
		// ecdsa signatures are non-deterministic, so rekor won't reject these as duplicates.
		sig, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
		if err != nil {
			return err
		}
		if _, err = submitRekorV2Entry(ctx, digest, sig, cert); err != nil {
			return err
		}
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
	readEntry, err := retrieveRekorV2Entry(ctx, privateKey, logIndex, treeSize)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(writtenEntry, readEntry) {
		return errors.New("submitted and retrieved entries do not match")
	}
	return nil
}

// rekorV2AndTSA sendsa request to the TSA and RekorV2, verifies the timestamp, and ensures that the entry can also be retrieved through RekorV2's read APIs.
func rekorV2AndTSA(ctx context.Context) error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		Logger.Fatalf("failed to generate key: %v", err)
	}

	artifact := []byte(time.Now().String())
	digest := sha256.Sum256(artifact)

	sig, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		return err
	}

	if err := submitAndVerifyTimestamp(ctx, sig); err != nil {
		return err
	}

	cert, err := fulcioWriteEndpoint(ctx, privateKey)
	if err != nil {
		return err
	}
	if err := submitAndRetrieveRekorV2Entry(ctx, privateKey, digest[:], sig, cert); err != nil {
		return err
	}
	return nil
}
