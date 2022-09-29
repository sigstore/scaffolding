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

package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/providers"
	"github.com/sigstore/fulcio/pkg/api"
	rclient "github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/models"
	hashedrekord "github.com/sigstore/rekor/pkg/types/hashedrekord/v0.0.1"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/sigstore/sigstore/pkg/signature"

	// Loads OIDC providers
	"github.com/sigstore/cosign/pkg/providers/all"
)

const (
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"

	fulcioEndpoint = "/api/v1/signingCert"
	rekorEndpoint  = "/api/v1/log/entries"
)

func setHeaders(req *http.Request, token string) {
	if token != "" {
		// Set the authorization header to our OIDC bearer token.
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Set the content-type to reflect we're sending JSON.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("Sigstore_Scaffolding_Prober/%s", versionInfo.GitVersion))
}

// fulcioWriteEndpoint tests the only write endpoint for Fulcio
// which is "/api/v1/signingCert", which requests a cert from Fulcio
func fulcioWriteEndpoint(ctx context.Context) error {
	if !all.Enabled(ctx) {
		return fmt.Errorf("no auth provider for fulcio is enabled")
	}
	tok, err := providers.Provide(ctx, "sigstore")
	if err != nil {
		return fmt.Errorf("getting provider: %w", err)
	}
	b, err := certificateRequest(ctx, tok)
	if err != nil {
		return fmt.Errorf("certificate response: %w", err)
	}

	// Construct the API endpoint for this handler
	endpoint := fulcioEndpoint
	hostPath := fulcioURL + endpoint

	req, err := http.NewRequest(http.MethodPost, hostPath, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	setHeaders(req, tok)

	t := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(t).Milliseconds()
	if err != nil {
		fmt.Printf("error requesting cert: %v\n", err.Error())
	}

	// Export data to prometheus
	exportDataToPrometheus(resp, fulcioURL, endpoint, POST, latency)
	return nil
}

// rekorWriteEndpoint tests the write endpoint for rekor, which is
// /api/v1/log/entries and adds an entry to the log
func rekorWriteEndpoint(ctx context.Context) error {
	endpoint := rekorEndpoint
	hostPath := rekorURL + endpoint

	body, err := rekorEntryRequest()
	if err != nil {
		return fmt.Errorf("rekor entry: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, hostPath, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	setHeaders(req, "")

	t := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(t).Milliseconds()
	exportDataToPrometheus(resp, rekorURL, endpoint, POST, latency)
	if err != nil {
		return fmt.Errorf("error adding entry: %w", err)
	}

	// If entry was added successfully, we should verify it
	rekorClient, err := rclient.GetRekorClient(rekorURL, rclient.WithUserAgent(fmt.Sprintf("Sigstore_Scaffolding_Prober/%s", versionInfo.GitVersion)))
	if err != nil {
		return fmt.Errorf("creating rekor client: %w", err)
	}
	defer resp.Body.Close()
	var logEntry models.LogEntry
	err = json.NewDecoder(resp.Body).Decode(&logEntry)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	var logEntryAnon models.LogEntryAnon
	for _, e := range logEntry {
		logEntryAnon = e
		break
	}
	verified := "true"
	if err = cosign.VerifyTLogEntry(ctx, rekorClient, &logEntryAnon); err != nil {
		verified = "false"
	}
	verificationCounter.With(prometheus.Labels{verifiedLabel: verified}).Inc()
	return err
}

func rekorEntryRequest() ([]byte, error) {
	payload := []byte(time.Now().String())
	priv, err := cosign.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generating keys: %w", err)
	}
	signer, err := signature.LoadECDSASignerVerifier(priv, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("loading signer verifier: %w", err)
	}
	pub, err := signer.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("public key: %w", err)
	}
	pubKey, err := cryptoutils.MarshalPublicKeyToPEM(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	sig, err := signer.SignMessage(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}

	h := sha256.Sum256(payload)
	e := &hashedrekord.V001Entry{
		HashedRekordObj: models.HashedrekordV001Schema{
			Data: &models.HashedrekordV001SchemaData{
				Hash: &models.HashedrekordV001SchemaDataHash{
					Algorithm: swag.String(models.HashedrekordV001SchemaDataHashAlgorithmSha256),
					Value:     swag.String(hex.EncodeToString(h[:])),
				},
			},
			Signature: &models.HashedrekordV001SchemaSignature{
				Content: strfmt.Base64(sig),
				PublicKey: &models.HashedrekordV001SchemaSignaturePublicKey{
					Content: strfmt.Base64(pubKey),
				},
			},
		},
	}
	pe := &models.Hashedrekord{
		APIVersion: swag.String(e.APIVersion()),
		Spec:       e.HashedRekordObj,
	}
	return json.Marshal(pe)
}

func certificateRequest(ctx context.Context, idToken string) ([]byte, error) {
	priv, err := cosign.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generating cert: %w", err)
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}

	tok, err := oauthflow.OIDConnect(defaultOIDCIssuer, defaultOIDCClientID, "", "", &oauthflow.StaticTokenGetter{RawToken: idToken})
	if err != nil {
		return nil, err
	}

	// Sign the email address as part of the request
	h := sha256.Sum256([]byte(tok.Subject))
	proof, err := ecdsa.SignASN1(rand.Reader, priv, h[:])
	if err != nil {
		return nil, err
	}

	cr := api.CertificateRequest{
		PublicKey: api.Key{
			Algorithm: "ecdsa",
			Content:   pubBytes,
		},
		SignedEmailAddress: proof,
	}

	return json.Marshal(cr)
}
