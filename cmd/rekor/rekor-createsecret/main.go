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
	"encoding/pem"
	"flag"
	"os"

	"github.com/sigstore/scaffolding/pkg/secret"
	"go.step.sm/crypto/pemutil"
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
)

var (
	signingSecretName  = flag.String("signing-secret", "rekor-signing-secret", "Secret to create the signing secret that Rekor will use.")
	signingKeyPassword = flag.String("signing-secret-pwd", "scaffoldtest", "Password to encrypt the signing secret with.")
	secretName         = flag.String("secret", "rekor-pub-key", "Secret to create the rekor public key in.")
)

func main() {
	flag.Parse()

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}

	ctx := signals.NewContext()
	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running rekor-createsecret Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get clientset: %v", err)
	}
	nsSecrets := clientset.CoreV1().Secrets(ns)

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
	// Encrypt the pem
	encryptedBlock, err := pemutil.EncryptPKCS8PrivateKey(rand.Reader, block.Bytes, []byte(*signingKeyPassword), x509.PEMCipherAES256)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to encrypt private key: %v", err)
	}

	privPEM := pem.EncodeToMemory(encryptedBlock)
	if privPEM == nil {
		logging.FromContext(ctx).Fatalf("Failed to encode encrypted private key: %v", err)
	}

	marshalledPubKey, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to marshal the public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: marshalledPubKey,
		},
	)

	signingSecretData := map[string][]byte{
		signingKeySecretKey:         privPEM,
		signingKeySecretPasswordKey: []byte(*signingKeyPassword),
	}

	if err := secret.ReconcileSecret(ctx, *signingSecretName, ns, signingSecretData, nsSecrets); err != nil {
		logging.FromContext(ctx).Fatalf("Unable to reconcile secret %s: %v", *signingSecretName, err)
	}

	publicKeyData := map[string][]byte{"public": pubPEM}
	if err := secret.ReconcileSecret(ctx, *secretName, ns, publicKeyData, nsSecrets); err != nil {
		logging.FromContext(ctx).Fatalf("Unable to reconcile secret %s: %v", *secretName, err)
	}
}
