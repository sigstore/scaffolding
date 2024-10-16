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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sigstore/scaffolding/pkg/secret"
	"go.step.sm/crypto/pemutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

var (
	secretName       = flag.String("secret", "fulcio-secrets", "Name of the secret to create for the certs")
	pubkeySecretName = flag.String("pubkeysecret", "fulcio-pub-key", "Name of the secret that holds the public Fulcio information like cert / public key")
	certOrg          = flag.String("cert-organization", "Linux Foundation", "Name of the organization for certificate creation")
	certCountry      = flag.String("cert-country", "USA", "Name of the country for certificate creation")
	certProvince     = flag.String("cert-province", "California", "Name of the province for certificate creation")
	certLocality     = flag.String("cert-locality", "San Francisco", "Name of the locality for certificate creation")
	certAddr         = flag.String("cert-address", "548 Market St", "Name of the address for certificate creation")
	certPostal       = flag.String("cert-postal", "57274", "Name of the postal code for certificate creation")
)

func main() {
	flag.Parse()

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}
	ctx := signals.NewContext()

	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running create_certs Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get clientset: %v", err)
	}

	// Just create the cert always in case we need to update or create it.
	certPEM, pubPEM, privPEM, pwd, err := createAll()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to create keys %v", err)
	}
	data := map[string][]byte{
		"cert":     certPEM,
		"private":  privPEM,
		"public":   pubPEM,
		"password": []byte(pwd),
	}

	// Reconcile the "main" secret that's used by Fulcio
	nsSecret := clientset.CoreV1().Secrets(ns)
	if err := secret.ReconcileSecret(ctx, *secretName, ns, data, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", ns, *secretName, err)
	}
	pubData := map[string][]byte{
		"cert":   certPEM,
		"public": pubPEM,
	}
	if err := secret.ReconcileSecret(ctx, *pubkeySecretName, ns, pubData, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", ns, *pubkeySecretName, err)
	}
}

// createAll creates a password protected keypair, and returns PEM encoded
// CA Cert, crypto.PublicKey, crypto.PrivateKey, password
func createAll() ([]byte, []byte, []byte, string, error) {
	// Generate ECDSA key.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to generate ecdsa key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to generate serial Number: %w", err)
	}
	rootCA := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{*certOrg},
			Country:       []string{*certCountry},
			Province:      []string{*certProvince},
			Locality:      []string{*certLocality},
			StreetAddress: []string{*certAddr},
			PostalCode:    []string{*certPostal},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		MaxPathLen:            1,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, rootCA, rootCA, privateKey.Public(), privateKey)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to create certificate: %w", err)
	}
	certPEM := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	)

	// Encode private key to PKCS #8 ASN.1 PEM.
	marshalledPrivKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("marshal pkcs8 private key: %w", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalledPrivKey,
	}

	// Generate a uuid as a password
	u := uuid.New()
	pwd := u.String()

	// Encrypt the pem
	block, err = pemutil.EncryptPKCS8PrivateKey(rand.Reader, block.Bytes, []byte(pwd), x509.PEMCipherAES256)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("EncryptPEMBlock failed: %w", err)
	}

	privPEM := pem.EncodeToMemory(block)
	if privPEM == nil {
		return nil, nil, nil, "", fmt.Errorf("EncodeToMemory private key failed: %w", err)
	}

	marshalledPubKey, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: marshalledPubKey,
		},
	)
	if pubPEM == nil {
		return nil, nil, nil, "", fmt.Errorf("EncodeToMemory public key failed: %w", err)
	}
	return certPEM, pubPEM, privPEM, pwd, nil
}
