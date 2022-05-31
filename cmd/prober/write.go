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
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/providers"
	"github.com/sigstore/fulcio/pkg/api"
	"github.com/sigstore/sigstore/pkg/oauthflow"
)

const (
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"
)

// fulcioWriteEndpoint tests the only write endpoint for Fulcio
// which is "/api/v1/signingCert", which requests a cert from Fulcio
func fulcioWriteEndpoint(ctx context.Context) error {
	if !providers.Enabled(ctx) {
		return fmt.Errorf("no auth provider for fulcio is enabled")
	}
	tok, err := providers.Provide(ctx, "sigstore")
	if err != nil {
		return errors.Wrap(err, "getting provider")
	}
	b, err := certificateRequest(ctx, tok)
	if err != nil {
		return errors.Wrap(err, "certificate response")
	}

	// Construct the API endpoint for this handler
	endpoint := "/api/v1/signingCert"
	hostPath := fulcioURL + endpoint

	req, err := http.NewRequest(http.MethodPost, hostPath, bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrap(err, "new request")
	}
	// Set the authorization header to our OIDC bearer token.
	req.Header.Set("Authorization", "Bearer "+tok)
	// Set the content-type to reflect we're sending JSON.
	req.Header.Set("Content-Type", "application/json")

	t := time.Now()
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(t).Milliseconds()
	if err != nil {
		fmt.Println("error requesting cert: ", err)
	}

	// Export data to prometheus
	statusCode := resp.StatusCode
	labels := prometheus.Labels{
		endpointLabel:   endpoint,
		statusCodeLabel: fmt.Sprintf("%d", statusCode),
		hostLabel:       fulcioURL,
	}
	endpointLatenciesSummary.With(labels).Observe(float64(latency))
	endpointLatenciesHistogram.With(labels).Observe(float64(latency))

	fmt.Println("Observing ", fulcioURL+endpoint)
	fmt.Println("Status code: ", statusCode)
	fmt.Println("Latency: ", latency)
	return nil
}

func certificateRequest(ctx context.Context, idToken string) ([]byte, error) {
	priv, err := cosign.GeneratePrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "generating cert")
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
