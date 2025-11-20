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
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/digitorus/timestamp"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag/conv"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/encoding/protojson"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/providers"
	common "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	rekor "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/rekor-tiles/pkg/generated/protobuf"
	"github.com/sigstore/rekor/pkg/generated/models"
	hashedrekord "github.com/sigstore/rekor/pkg/types/hashedrekord/v0.0.1"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/oauthflow"
	"github.com/sigstore/sigstore/pkg/signature"

	// Loads OIDC providers
	"github.com/sigstore/cosign/v2/pkg/providers/all"
)

const (
	defaultOIDCIssuer   = "https://oauth2.sigstore.dev/auth"
	defaultOIDCClientID = "sigstore"

	fulcioEndpoint       = "/api/v2/signingCert"
	fulcioLegacyEndpoint = "/api/v1/signingCert"
	rekorEndpoint        = "/api/v1/log/entries"
	rekorV2Endpoint      = "/api/v2/log/entries"
)

func setHeaders(req *retryablehttp.Request, token string, rpc ReadProberCheck) {
	if token != "" {
		// Set the authorization header to our OIDC bearer token.
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Set the content-type to reflect we're sending JSON.
	switch rpc.Accept {
	case "":
		req.Header.Set("Accept", "application/json")
	default:
		req.Header.Set("Accept", rpc.Accept)
	}
	if req.ContentLength > 0 {
		switch rpc.ContentType {
		case "":
			req.Header.Set("Content-Type", "application/json")
		default:
			req.Header.Set("Content-Type", rpc.ContentType)
		}
	}
	req.Header.Set("User-Agent", fmt.Sprintf("Sigstore_Scaffolding_Prober/%s", versionInfo.GitVersion))
	// Set this value (even though it is not coming through an GCP LB) to correlate prober req/response
	req.Header.Set("X-Cloud-Trace-Context", uuid.Must(uuid.NewV7()).String())
}

// fulcioWriteLegacyEndpoint tests the /api/v1/signingCert write endpoint for Fulcio.
func fulcioWriteLegacyEndpoint(ctx context.Context, priv *ecdsa.PrivateKey, fulcioService root.Service) (*x509.Certificate, error) {
	if !all.Enabled(ctx) {
		return nil, fmt.Errorf("no auth provider for fulcio is enabled")
	}
	tok, err := providers.Provide(ctx, "sigstore")
	if err != nil {
		return nil, fmt.Errorf("getting provider: %w", err)
	}
	b, err := legacyCertificateRequest(ctx, tok, priv)
	if err != nil {
		return nil, fmt.Errorf("certificate response: %w", err)
	}

	// Construct the API endpoint for this handler
	endpoint := fulcioLegacyEndpoint
	hostPath := fulcioService.URL + endpoint

	req, err := retryablehttp.NewRequest(http.MethodPost, hostPath, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	setHeaders(req, tok, ReadProberCheck{})

	t := time.Now()
	resp, err := retryableClient.Do(req)
	latency := time.Since(t).Milliseconds()
	if err != nil {
		Logger.Errorf("error requesting cert: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid status code '%s' when requesting a cert from Fulcio with body '%s'", resp.Status, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger.Errorf("error reading response from Fulcio: %v", err)
		return nil, err
	}
	certBlock, chainPEM := pem.Decode(responseBody)
	if certBlock == nil || chainPEM == nil {
		Logger.Errorf("did not find expected certificates")
	}
	intermediateBlock, rootPEM := pem.Decode(chainPEM)
	if intermediateBlock == nil || rootPEM == nil {
		Logger.Errorf("did not find expected certificate chain in response from Fulcio")
	}
	certPEM := pem.EncodeToMemory(certBlock)
	cert, err := cryptoutils.UnmarshalCertificatesFromPEM(certPEM)
	if err != nil {
		Logger.Errorf("error unmarshalling leaf certificate from Fulcio: %v", err)
		return nil, err
	}
	if len(cert) != 1 {
		Logger.Errorf("unexpected number of certificates after unmarshalling got %d, expected 1", len(cert))
		return nil, err
	}

	// Export data to prometheus
	exportDataToPrometheus(resp, fulcioService.URL, endpoint, POST, latency)
	return cert[0], nil
}

// fulcioWriteEndpoint tests the /api/v2/signingCert write endpoint for Fulcio.
func fulcioWriteEndpoint(ctx context.Context, priv *ecdsa.PrivateKey, fulcioService root.Service, trustedRoot *root.TrustedRoot) (*x509.Certificate, error) {
	if !all.Enabled(ctx) {
		return nil, fmt.Errorf("no auth provider for fulcio is enabled")
	}
	tok, err := providers.Provide(ctx, "sigstore")
	if err != nil {
		return nil, fmt.Errorf("getting provider: %w", err)
	}
	b, err := certificateRequest(ctx, tok, priv)
	if err != nil {
		return nil, fmt.Errorf("certificate response: %w", err)
	}

	// Construct the API endpoint for this handler
	endpoint := fulcioEndpoint
	hostPath := fulcioService.URL + endpoint

	req, err := retryablehttp.NewRequest(http.MethodPost, hostPath, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	setHeaders(req, tok, ReadProberCheck{})

	t := time.Now()
	resp, err := retryableClient.Do(req)
	latency := time.Since(t).Milliseconds()
	if err != nil {
		Logger.Errorf("error requesting cert: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid status code '%s' when requesting a cert from Fulcio with body '%s'", resp.Status, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger.Errorf("error reading response from Fulcio: %v", err)
		return nil, err
	}
	var fulcioResp SigningCertificateResponse
	if err := json.Unmarshal(responseBody, &fulcioResp); err != nil {
		Logger.Errorf("error parsing response from Fulcio: %v", err)
		return nil, err
	}

	var activeCA *root.FulcioCertificateAuthority
	now := time.Now()
	for _, ca := range trustedRoot.FulcioCertificateAuthorities() {
		if fulcioCA, ok := ca.(*root.FulcioCertificateAuthority); ok {
			isActive := now.After(fulcioCA.ValidityPeriodStart) && (fulcioCA.ValidityPeriodEnd.IsZero() || now.Before(fulcioCA.ValidityPeriodEnd))
			if fulcioCA.URI == fulcioService.URL && isActive {
				activeCA = fulcioCA
				break
			}
		}
	}
	if activeCA == nil {
		return nil, fmt.Errorf("could not find an active Fulcio CA for URI %s in trusted root", fulcioService.URL)
	}

	fulcioExpectedCertCount := len(activeCA.Intermediates) + 2 // leaf + intermediates + root
	fulcioRespCertCount := len(fulcioResp.CertificatesWithSct.CertificateChain.Certificates)
	if fulcioRespCertCount != fulcioExpectedCertCount {
		err := fmt.Errorf("unexpected number of certificates, got %d, expected %d", fulcioRespCertCount, fulcioExpectedCertCount)
		return nil, err
	}

	cert, err := cryptoutils.UnmarshalCertificatesFromPEM([]byte(fulcioResp.CertificatesWithSct.CertificateChain.Certificates[0]))
	if err != nil {
		Logger.Errorf("error unmarshalling leaf certificate from Fulcio: %v", err)
		return nil, err
	}
	if len(cert) != 1 {
		Logger.Errorf("unexpected number of certificates after unmarshalling got %d, expected 1", len(cert))
		return nil, err
	}

	// Export data to prometheus
	exportDataToPrometheus(resp, fulcioService.URL, endpoint, POST, latency)
	return cert[0], nil
}

func makeRekorV1Request(cert *x509.Certificate, priv *ecdsa.PrivateKey, hostPath string) (*http.Response, int64, error) {
	body, err := rekorV1EntryRequest(cert, priv)
	if err != nil {
		return nil, -1, fmt.Errorf("rekor entry: %w", err)
	}
	req, err := retryablehttp.NewRequest(http.MethodPost, hostPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, -1, fmt.Errorf("new request: %w", err)
	}
	setHeaders(req, "", ReadProberCheck{})

	t := time.Now()
	resp, err := retryableClient.Do(req)
	latency := time.Since(t).Milliseconds()
	return resp, latency, err
}

// rekorV1WriteEndpoint tests the write endpoint for rekor v1, which is
// /api/v1/log/entries and adds an entry to the log
// if a certificate is provided, the Rekor entry will contain that certificate,
// otherwise the provided key is used
func rekorV1WriteEndpoint(ctx context.Context, cert *x509.Certificate, priv *ecdsa.PrivateKey, rekorV1Services []root.Service, trustedRoot *root.TrustedRoot) error {
	var lastErr error
	for _, s := range rekorV1Services {
		verified := "false"
		endpoint := rekorEndpoint
		hostPath := s.URL + endpoint
		defer func() {
			verificationCounter.With(prometheus.Labels{verifiedLabel: verified}).Inc()
		}()
		var resp *http.Response
		var latency int64
		var err error
		// A new body should be created when it is conflicted
		for i := 1; i < 10; i++ {
			resp, latency, err = makeRekorV1Request(cert, priv, hostPath)
			if err != nil {
				lastErr = fmt.Errorf("error adding entry: %w", err)
				break
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusConflict {
				break
			}
		}
		if err != nil || resp == nil {
			continue
		}
		exportDataToPrometheus(resp, s.URL, endpoint, POST, latency)

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("invalid status code '%s' when creating entry in rekor: '%s'", resp.Status, string(body))
			continue
		}
		// If entry was added successfully, we should verify it
		var logEntry models.LogEntry
		err = json.NewDecoder(resp.Body).Decode(&logEntry)
		if err != nil {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("error decoding the log entry with body '%s' and error: %w", string(body), err)
			continue
		}
		var logEntryAnon models.LogEntryAnon
		for _, e := range logEntry {
			logEntryAnon = e
			break
		}
		if err = cosign.VerifyTLogEntryOffline(ctx, &logEntryAnon, nil, trustedRoot); err == nil {
			verified = "true"
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func rekorV1EntryRequest(cert *x509.Certificate, priv *ecdsa.PrivateKey) ([]byte, error) {
	// sign payload
	payload := []byte(time.Now().String())
	signer, err := signature.LoadECDSASignerVerifier(priv, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("loading signer verifier: %w", err)
	}
	sig, err := signer.SignMessage(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}

	var verifier []byte
	if cert != nil {
		certPEM, err := cryptoutils.MarshalCertificateToPEM(cert)
		if err != nil {
			return nil, fmt.Errorf("error marshalling certificate: %w", err)
		}
		verifier = certPEM
	} else {
		pubKeyPEM, err := cryptoutils.MarshalPublicKeyToPEM(priv.Public())
		if err != nil {
			return nil, fmt.Errorf("error marshalling public key: %w", err)
		}
		verifier = pubKeyPEM
	}

	h := sha256.Sum256(payload)
	e := &hashedrekord.V001Entry{
		HashedRekordObj: models.HashedrekordV001Schema{
			Data: &models.HashedrekordV001SchemaData{
				Hash: &models.HashedrekordV001SchemaDataHash{
					Algorithm: conv.Pointer(models.HashedrekordV001SchemaDataHashAlgorithmSha256),
					Value:     conv.Pointer(hex.EncodeToString(h[:])),
				},
			},
			Signature: &models.HashedrekordV001SchemaSignature{
				Content: strfmt.Base64(sig),
				PublicKey: &models.HashedrekordV001SchemaSignaturePublicKey{
					Content: strfmt.Base64(verifier),
				},
			},
		},
	}
	pe := &models.Hashedrekord{
		APIVersion: conv.Pointer(e.APIVersion()),
		Spec:       e.HashedRekordObj,
	}
	return json.Marshal(pe)
}

// rekorV2WriteEndpoint tests the write endpoint for rekor v2, which is
// /api/v2/log/entries and adds an entry to the log
// if a certificate is provided, the Rekor entry will contain that certificate,
// otherwise the provided key is used
func rekorV2WriteEndpoint(ctx context.Context, cert *x509.Certificate, priv *ecdsa.PrivateKey, rekorV2Services []root.Service) error {
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
		KeyDetails: common.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256,
	}
	if cert != nil {
		verifier.Verifier = &protobuf.Verifier_X509Certificate{
			X509Certificate: &common.X509Certificate{
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
		Endpoint: rekorV2Endpoint,
		Method:   POST,
		Body:     reqBytes,
	}
	var lastErr error
	for _, rekorV2Service := range rekorV2Services {
		respBytes, err := observeRequest(rekorV2Service.URL, proberCheck)
		if err != nil {
			lastErr = err
			continue
		}
		tle := rekor.TransparencyLogEntry{}
		if err := protojson.Unmarshal(respBytes, &tle); err != nil {
			lastErr = err
			continue
		}
		tleBody := protobuf.Entry{}
		if err := protojson.Unmarshal(tle.CanonicalizedBody, &tleBody); err != nil {
			lastErr = err
			continue
		}
		// basic content matching, without proof or signature verification.
		tleBodyDigest := tleBody.Spec.GetHashedRekordV002().Data.Digest
		if !bytes.Equal(tleBodyDigest, digest[:]) {
			lastErr = fmt.Errorf("tleEntry digest does not match: got: %s, want: %s", tleBodyDigest, digest)
			continue
		}
		tleEntrySig := tleBody.Spec.GetHashedRekordV002().Signature.Content
		if !bytes.Equal(tleEntrySig, sig) {
			lastErr = fmt.Errorf("tleEntry signature does not match: got: %s, want: %s", tleEntrySig, sig)
			continue
		}
		return nil
	}
	return lastErr
}

func tsaWriteEndpoint(ctx context.Context, priv *ecdsa.PrivateKey, tsaServices []root.Service, trustedRoot *root.TrustedRoot) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	artifact := []byte(time.Now().String())
	digest := sha256.Sum256(artifact)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		return err
	}

	signatureHash := sha256.Sum256(sig)

	getTSReq := &timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: signatureHash[:],
	}

	getTSReqBytes, err := getTSReq.Marshal()
	if err != nil {
		return fmt.Errorf("marshalling the timestamp request: %w", err)
	}

	proberCheck := TSAEndpoints[0]
	proberCheck.Body = getTSReqBytes

	var lastErr error
	for _, tsaService := range tsaServices {
		getTSRespBytes, err := observeRequest(tsaService.URL, proberCheck)
		if err != nil {
			lastErr = err
			continue
		}
		for _, tsa := range trustedRoot.TimestampingAuthorities() {
			if _, err := tsa.Verify(getTSRespBytes, sig); err == nil {
				return nil
			}
			lastErr = err
		}
	}
	if lastErr != nil {
		return fmt.Errorf("verifying the timestamp: %w", lastErr)
	}

	return errors.New("no trusted timestamp authority was able to verify the timestamp")
}

func certificateRequest(_ context.Context, idToken string, priv *ecdsa.PrivateKey) ([]byte, error) {
	pubBytesPEM, err := cryptoutils.MarshalPublicKeyToPEM(priv.Public())
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

	req := SigningCertificateRequest{
		PublicKeyRequest: PublicKeyRequest{
			PublicKey: PublicKey{
				Content: string(pubBytesPEM),
			},
			ProofOfPossession: proof,
		},
	}

	return json.Marshal(req)
}

func legacyCertificateRequest(_ context.Context, idToken string, priv *ecdsa.PrivateKey) ([]byte, error) {
	pubBytesPEM, err := cryptoutils.MarshalPublicKeyToPEM(priv.Public())
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

	req := SigningCertificateRequestLegacy{
		PublicKey: PublicKeyLegacy{
			Content: pubBytesPEM,
		},
		SignedEmailAddress: proof,
	}

	return json.Marshal(req)
}

type SigningCertificateRequest struct {
	PublicKeyRequest PublicKeyRequest `json:"publicKeyRequest"`
}

type SigningCertificateRequestLegacy struct {
	PublicKey          PublicKeyLegacy `json:"publicKey"`
	SignedEmailAddress []byte          `json:"signedEmailAddress"`
}

type SigningCertificateResponse struct {
	CertificatesWithSct SignedCertificateEmbeddedSct `json:"signedCertificateEmbeddedSct"`
}

type SignedCertificateEmbeddedSct struct {
	CertificateChain CertificateChain `json:"chain"`
}

type CertificateChain struct {
	Certificates []string `json:"certificates"`
}

type PublicKeyRequest struct {
	PublicKey         PublicKey `json:"publicKey"`
	ProofOfPossession []byte    `json:"proofOfPossession"`
}

type PublicKey struct {
	Content string `json:"content"`
}

type PublicKeyLegacy struct {
	Content []byte `json:"content"`
}
