// Copyright 2022 The Sigstore Authors
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

package ctlog

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/certificate-transparency-go/trillian/ctfe/configpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sigstore/rekor/pkg/pki/x509/testutils"
	"google.golang.org/protobuf/encoding/prototext"
)

// Just a test Root Cert from a Fulcio instance spun up using Scaffolding.
const (
	existingRootCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
`
	testConfig = "YmFja2VuZHM6e2JhY2tlbmQ6e25hbWU6InRyaWxsaWFuIn19ICBsb2dfY29uZmlnczp7Y29uZmlnOntsb2dfaWQ6MjAyMiAgcHJlZml4OiIyMDIyLWN0bG9nIiAgcm9vdHNfcGVtX2ZpbGU6Ii9jdGZlLWtleXMvZnVsY2lvLTAiICBwcml2YXRlX2tleTp7W3R5cGUuZ29vZ2xlYXBpcy5jb20va2V5c3BiLlBFTUtleUZpbGVdOntwYXRoOiIvY3RmZS1rZXlzL3ByaXZrZXkucGVtIiAgcGFzc3dvcmQ6Im15dGVzdHBhc3N3b3JkIn19ICBwdWJsaWNfa2V5OntkZXI6IjBZMFx4MTNceDA2XHgwNypceDg2SFx4Y2U9XHgwMlx4MDFceDA2XHgwOCpceDg2SFx4Y2U9XHgwM1x4MDFceDA3XHgwM0JceDAwXHgwNNWwXHhlM1x4YTZYXHhjZS9ceGE1XHg5NFx4ZjZceGM2Plx4ODJceGJje1x4ZGVceGYwfG0rXHhkMVx4Y2U7XHg4NVx4YmZceGYyXHhmOFx4OTRceGYwfVx4ZDlceDFkPlx4N2ZKKFx4YzY+cVx4OGZceGM4XHgwZVx4YTJdXHgxNFx4ODhceGM4XHhkNX7Du2ZzXHhlZVx4OTlceDFicVx4MGVgR1x4ZWZceGUyQlx4ZjQifSAgZXh0X2tleV91c2FnZXM6IkNvZGVTaWduaW5nIiAgbG9nX2JhY2tlbmRfbmFtZToidHJpbGxpYW4ifX0="

	privateKeyEncoded = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tClByb2MtVHlwZTogNCxFTkNSWVBURUQKREVLLUluZm86IEFFUy0yNTYtQ0JDLDJiNDU2MGUyY2RlMGE3ZWM0NjZlMzkzYWRmYmE0Y2I0CiAgICAgICAKVUk4d2lUbXhNajhKWXVHSUFEMnpKVjRmQjZHUE9wUGhxSldYdlR3RWFucHBzTXN3UUFCaVZ5NWdkSi9BNThQVAo0ZTFFSDM4Y3Z3YTBMQjQ2SHBoZW9vWCtJM2RHdHlzRUpFR0d3QXMwYUhkU25aeVV3TnRpalRUQkZJcWxzd3pKCnI2WmJ4dmlxZVRmRm80ZUtEMGorRjlja2R3d2dGT2YzRHdaUUMrNEN1cVNqczdaZkFKZEF6Lys0c2JRd1ZzQUIKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo="

	publicKeyEncoded = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFT3Y1bzVXV0tZaVVSODdzNGZpMEpKbU1EUVV2cQpSck1mNGRlQnpzV3BCWVdVK1Y4TXVDMkh6aTFOTHI4czRlQ0J5dWVDZmFQWFN4STgzUkowamEwbnd3PT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="
)

func TestUnmarshal(t *testing.T) {
	in, err := createBaseConfig(t)
	if err != nil {
		t.Fatalf("failed to createBaseConfig: %v", err)
	}
	config, err := Unmarshal(context.Background(), in)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	t.Logf("Got: %s", config.String())
	if len(config.FulcioCerts) != 1 || bytes.Compare(config.FulcioCerts[0], []byte(existingRootCert)) != 0 {
		t.Errorf("Fulciosecrets differ")
	}
}

func TestRoundTrip(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Private Key: %v", err)
	}

	configIn := &CTLogConfig{
		PrivKey:         privateKey,
		PrivKeyPassword: "mytestpassword",
		PubKey:          privateKey.Public().(*ecdsa.PublicKey),
		LogID:           2022,
		LogPrefix:       "2022-ctlog",
	}
	configIn.FulcioCerts = append(configIn.FulcioCerts, []byte(existingRootCert))

	marshaledConfig, err := configIn.MarshalConfig(context.Background())
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	configOut, err := Unmarshal(context.Background(), marshaledConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if !reflect.DeepEqual(configIn, configOut) {
		t.Errorf("Things differ=%s", cmp.Diff(configIn, configOut, cmpopts.IgnoreUnexported(CTLogConfig{})))
	}

	if configIn.PrivKey == nil || configOut.PrivKey == nil || !configOut.PrivKey.Equal(configIn.PrivKey) {
		t.Errorf("Private Keys differ")
	}
}

func TestAddNewFulcioAndRemoveOld(t *testing.T) {
	ctx := context.TODO()
	in, err := createBaseConfig(t)
	if err != nil {
		t.Fatalf("failed to createBaseConfig: %v", err)
	}
	config, err := Unmarshal(ctx, in)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	newFulcioCert, err := createTestCert(t)
	if err != nil {
		t.Fatalf("Failed to create a test certificate: %v", err)
	}
	config.AddFulcioRoot(ctx, newFulcioCert)
	marshaled, err := config.MarshalConfig(context.Background())
	if err != nil {
		t.Fatalf("Failed to MarshalConfig: %v", err)
	}

	// Now test that we have configuration that trusts both Fulcio roots
	// simulating while one is being spun down.
	expected := [][]byte{}
	expected = append(expected, []byte(existingRootCert), newFulcioCert)
	validateFulcioEntries(ctx, marshaled, expected, t)

	newConfig, err := Unmarshal(ctx, marshaled)
	if len(newConfig.FulcioCerts) != 2 {
		t.Fatalf("Unexpected number of FulcioCerts, got %d", len(newConfig.FulcioCerts))
	}

	// Now for our next trick, pretend we're rotating, so take out the
	// existing entry from the trusted certs.
	newConfig.RemoveFulcioRoot(ctx, []byte(existingRootCert))
	marshaledNew, err := newConfig.MarshalConfig(context.Background())
	if err != nil {
		t.Fatalf("Failed to marshal new configuration after removal: %v", err)
	}

	// Now test that we have configuration that trusts only the new Fulcio
	// root, simulating that the old one has been spun down.
	expected = make([][]byte, 0)
	expected = append(expected, []byte(newFulcioCert))
	validateFulcioEntries(ctx, marshaledNew, expected, t)
}

func createBaseConfig(t *testing.T) (map[string][]byte, error) {
	t.Helper()
	c, err := b64.StdEncoding.DecodeString(testConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode testConfig: %w", err)
	}
	private, err := b64.StdEncoding.DecodeString(privateKeyEncoded)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode privateKeyEncoded: %w", err)
	}
	public, err := b64.StdEncoding.DecodeString(publicKeyEncoded)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode publicKeyEncoded: %w", err)
	}
	return map[string][]byte{
		"config":  c,
		"private": private,
		"public":  public,
		"rootca":  []byte(existingRootCert),
	}, nil
}

func createTestCert(t *testing.T) ([]byte, error) {
	// Generate x509 Root CA
	rootCA, rootKey, err := testutils.GenerateRootCa()
	if err != nil {
		return nil, fmt.Errorf("GenerateKey failed: %w", err)
	}
	// Extract public component.
	pub := rootKey.Public()

	derBytes, err := x509.CreateCertificate(rand.Reader, rootCA, rootCA, pub, rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	return pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	), nil
}

// validateFulcioEntries will take in a marshalled config and validate
// that it has only the fulcioCerts specified as well as matching number
// of fulcio-%d entries in both the CTLog configuration as well as in the
// passed in map (that gets mounted as secret).
func validateFulcioEntries(ctx context.Context, config map[string][]byte, fulcioCerts [][]byte, t *testing.T) {
	t.Helper()
	// This keeps track of if we've seen a file entry in the CTLog config
	// for fulcio-%d entry. There should be one for each fulcioCerts
	foundFile := make(map[string]bool, len(fulcioCerts))
	for i := range fulcioCerts {
		foundFile[fmt.Sprintf("%sfulcio-%d", rootsPemFileDir, i)] = false
	}
	foundPEM := make([]bool, len(fulcioCerts))
	PEMEntriesFound := 0

	// First make sure we have all the PEMs that we expect in the map.
	for k, v := range config {
		if strings.HasPrefix(k, "fulcio-") {
			for i, fulcioCert := range fulcioCerts {
				if bytes.Compare(v, fulcioCert) == 0 {
					foundPEM[i] = true
				}
			}
			PEMEntriesFound++
		}
	}

	if PEMEntriesFound != len(fulcioCerts) {
		t.Errorf("Unexpected number of PEM entries, want: %d got %d", len(fulcioCerts), PEMEntriesFound)
	}
	for i, found := range foundPEM {
		if !found {
			t.Errorf("Failed to find a PEM for entry %d", i)
		}
	}

	// Then validate that for each of those there's an entry in the CTLog
	// config file.
	// Then check the log config that it tells CTLog to trust these two certs
	// above
	multiConfig := configpb.LogMultiConfig{}
	if err := prototext.Unmarshal(config[ConfigKey], &multiConfig); err != nil {
		t.Fatalf("failed to unmarshal ctlog proto: %v", err)
	}

	trustedCerts := multiConfig.GetLogConfigs().Config[0].RootsPemFile
	if len(trustedCerts) != len(fulcioCerts) {
		t.Fatalf("Unexpected number of file entries, want: %d got %d", len(fulcioCerts), len(trustedCerts))
	}
	for _, fileName := range trustedCerts {
		if strings.HasPrefix(fileName, rootsPemFileDir) {
			foundFile[fileName] = true
		}
	}
	for fileName, found := range foundFile {
		if !found {
			t.Errorf("Failed to find a PEM for entry %s", fileName)
		}
	}
}
