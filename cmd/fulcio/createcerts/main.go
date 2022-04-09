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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

const (
	// Key in the configmap holding the value of the tree.
	bitSize = 4096
)

var (
	secretName   = flag.String("secret", "fulcio-secrets", "Name of the secret to create for the certs")
	certOrg      = flag.String("cert-organization", "Linux Foundation", "Name of the organization for certificate creation")
	certCountry  = flag.String("cert-country", "USA", "Name of the country for certificate creation")
	certProvince = flag.String("cert-province", "California", "Name of the province for certificate creation")
	certLocality = flag.String("cert-locality", "San Francisco", "Name of the locality for certificate creation")
	certAddr     = flag.String("cert-address", "548 Market St", "Name of the address for certificate creation")
	certPostal   = flag.String("cert-postal", "57274", "Name of the postal code for certificate creation")
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
	data := make(map[string][]byte)
	data["cert"] = certPEM
	data["private"] = privPEM
	data["public"] = pubPEM
	data["password"] = []byte(pwd)

	// See if there's an existing secret first
	existingSecret, err := clientset.CoreV1().Secrets(ns).Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *secretName, err)
	}

	// If we found the secret, just make sure all the fields are there.
	if err == nil && existingSecret != nil {
		_, certok := existingSecret.Data["cert"]
		_, privok := existingSecret.Data["private"]
		_, pubok := existingSecret.Data["public"]
		_, pwdok := existingSecret.Data["password"]

		if privok && pubok && pwdok && certok {
			logging.FromContext(ctx).Infof("Found existing secret config with all the keys")
			return
		}
		existingSecret.Data = data
		_, err = clientset.CoreV1().Secrets(ns).Update(ctx, existingSecret, metav1.UpdateOptions{})
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to update secret %s/%s: %v", ns, *secretName, err)
		}
		logging.FromContext(ctx).Infof("Updated secret %s/%s", ns, *secretName)
		return
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      *secretName,
		},
		Data: data,
	}
	_, err = clientset.CoreV1().Secrets(ns).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to create secret %s/%s: %v", ns, *secretName, err)
	}
	logging.FromContext(ctx).Infof("Created secret %s/%s", ns, *secretName)
}

// createAll creates a password protected keypair, and returns PEM encoded
// CA Cert, crypto.PublicKey, crypto.PrivateKey, password
func createAll() ([]byte, []byte, []byte, string, error) {
	// Generate RSA key.
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "GenerateKey failed")
	}
	// Extract public component.
	pub := key.Public()

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "failed to generate serial Number")
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
	derBytes, err := x509.CreateCertificate(rand.Reader, rootCA, rootCA, pub, key)
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "failed to create certificate")
	}
	certPEM := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	)

	// Encode private key to PKCS#1 ASN.1 PEM.
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	// Generate a uuid as a password
	u := uuid.New()
	pwd := u.String()

	// Encrypt the pem
	block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(pwd), x509.PEMCipherAES256) // nolint
	if err != nil {
		return nil, nil, nil, "", errors.Wrap(err, "EncryptPEMBlock failed")
	}

	privPEM := pem.EncodeToMemory(block)
	if privPEM == nil {
		return nil, nil, nil, "", errors.New("EncodeToMemory private key failed")
	}
	// Encode public key to PKCS#1 ASN.1 PEM.
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
		},
	)
	if pubPEM == nil {
		return nil, nil, nil, "", errors.New("EncodeToMemory public key failed")
	}
	return certPEM, pubPEM, privPEM, pwd, nil
}
