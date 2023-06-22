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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"os"

	"github.com/google/uuid"

	"github.com/sigstore/scaffolding/pkg/secret"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/timestamp-authority/pkg/signer"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

const (
	// Key created in the secret holding the encrypted private key.
	signingKeySecretKey = "signing-secret"
	// Key created in the secret holding the encrypt private key passord.
	signingKeySecretPasswordKey = "signing-secret-password"
	// Key created in the secret holding the certchain.
	certChainKey = "cert-chain"
)

var (
	secretName = flag.String("secret", "tsa-cert-chain", "Name of the secret to create for the cert chain and private key")
)

func main() {
	flag.Parse()

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}
	ctx := signals.NewContext()
	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running createcertchain Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get clientset: %v", err)
	}
	nsSecrets := clientset.CoreV1().Secrets(ns)
	existing, err := nsSecrets.Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Fatalf("Failed to get secret: %v", err)
	}
	if existing.Data != nil &&
		existing.Data[certChainKey] != nil &&
		existing.Data[signingKeySecretKey] != nil &&
		existing.Data[signingKeySecretPasswordKey] != nil {
		logging.FromContext(ctx).Info("Found existing secret, pwd, and certchain in secret")
		return
	}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to generate private key: %v", err)
	}
	// Encode private key to PKCS #8 ASN.1 PEM.
	marshalledPrivKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to marshal private key: %v", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalledPrivKey,
	}
	// Encrypt the pem with a uuid
	pwd := uuid.New().String()
	encryptedBlock, err := x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(pwd), x509.PEMCipherAES256) // nolint
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to encrypt private key: %v", err)
	}

	privPEM := pem.EncodeToMemory(encryptedBlock)
	if privPEM == nil {
		logging.FromContext(ctx).Fatalf("Failed to encode encrypted private key: %v", err)
	}
	secretData := make(map[string][]byte, 3)
	secretData[signingKeySecretKey] = privPEM
	secretData[signingKeySecretPasswordKey] = []byte(pwd)

	s, err := signature.LoadECDSASignerVerifier(privateKey, crypto.SHA256)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to create signer: %v", err)
	}

	chain, err := signer.NewTimestampingCertWithChain(s)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to create cert chain: %v", err)
	}

	chainBytes, err := cryptoutils.MarshalCertificatesToPEM(chain)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to marshal certchain to PEM: %v", err)
	}

	secretData[certChainKey] = chainBytes
	err = secret.ReconcileSecret(ctx, *secretName, ns, secretData, nsSecrets)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to reconcile secret: %v", err)
	}
}
