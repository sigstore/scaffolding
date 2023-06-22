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
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	fulcioclient "github.com/sigstore/fulcio/pkg/api"
	"github.com/sigstore/scaffolding/pkg/ctlog"
	"github.com/sigstore/scaffolding/pkg/secret"
	apierrs "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

const (
	// Key in the configmap holding the value of the tree.
	treeKey    = "treeID"
	configKey  = "config"
	privateKey = "private"
	publicKey  = "public"
	bitSize    = 4096
)

var (
	cmname             = flag.String("configmap", "ctlog-config", "Name of the configmap where the treeID lives")
	privateKeySecret   = flag.String("private-secret", "", "If there's an existing private key that should be used, read it from this secret, decrypt with the key-password and use it instead of creating a new one.")
	secretName         = flag.String("secret", "ctlog-secrets", "Name of the secret to create for the keyfiles")
	pubKeySecretName   = flag.String("pubkeysecret", "ctlog-public-key", "Name of the secret to create containing only the public key")
	ctlogPrefix        = flag.String("log-prefix", "sigstorescaffolding", "Prefix to append to the url. This is basically the name of the log.")
	fulcioURL          = flag.String("fulcio-url", "http://fulcio.fulcio-system.svc", "Where to fetch the fulcio Root CA from")
	trillianServerAddr = flag.String("trillian-server", "log-server.trillian-system.svc:80", "Address of the gRPC Trillian Admin Server (host:port)")
	// TODO: Support ed25519
	keyType     = flag.String("keytype", "ecdsa", "Which private key to generate [rsa,ecdsa]")
	curveType   = flag.String("curvetype", "p256", "Curve type to use [p256, p384,p521]")
	keyPassword = flag.String("key-password", "test", "Password for encrypting the PEM key")

	// Supported elliptic curve functions.
	supportedCurves = map[string]elliptic.Curve{
		"p256": elliptic.P256(),
		"p384": elliptic.P384(),
		"p521": elliptic.P521(),
	}
)

func main() {
	flag.Parse()
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}

	if *keyType != "rsa" && *keyType != "ecdsa" {
		panic(fmt.Sprintf("invalid keytype specified: %s, support for [rsa,ecdsa]", *keyType))
	}

	if _, ok := supportedCurves[*curveType]; !ok {
		panic(fmt.Sprintf("invalid curvetype specified: %s, support for [p256,p384,p521]", *keyType))
	}
	ctx := signals.NewContext()

	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running create_ct_config Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get clientset: %v", err)
	}
	cm, err := clientset.CoreV1().ConfigMaps(ns).Get(ctx, *cmname, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get the configmap %s/%s: %v", ns, *cmname, err)
	}

	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	treeID, ok := cm.Data[treeKey]
	if !ok {
		logging.FromContext(ctx).Errorf("No treeid yet, bailing")
		os.Exit(-1)
	}

	logging.FromContext(ctx).Infof("Found treeid: %s", treeID)
	treeIDInt, err := strconv.ParseInt(treeID, 10, 64)
	if err != nil {
		logging.FromContext(ctx).Panicf("Invalid TreeID %s : %v", treeID, err)
	}

	// Fetch the fulcio Root CA
	u, err := url.Parse(*fulcioURL)
	if err != nil {
		logging.FromContext(ctx).Panicf("Invalid fulcioURL %s : %v", *fulcioURL, err)
	}
	client := fulcioclient.NewClient(u)
	root, err := client.RootCert()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to fetch fulcio Root cert: %w", err)
	}

	// See if there's an existing configuration already in the ConfigMap
	var existingCMConfig []byte
	if cm.BinaryData != nil && cm.BinaryData[configKey] != nil {
		logging.FromContext(ctx).Infof("Found existing ctlog config in ConfigMap")
		existingCMConfig = cm.BinaryData[configKey]
	}

	// See if there's existing secret with the keys we want
	nsSecret := clientset.CoreV1().Secrets(ns)
	existingSecret, err := nsSecret.Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Fatalf("Failed to get secret %s/%s: %v", ns, *secretName, err)
	}

	// If any of the private, public or config either from secret or configmap
	// is not there, create a new configuration.
	if existingSecret.Data[privateKey] == nil ||
		existingSecret.Data[publicKey] == nil ||
		(existingSecret.Data[configKey] == nil && existingCMConfig == nil) {
		var ctlogConfig *ctlog.Config
		var err error
		if *privateKeySecret != "" {
			// We have an existing private key, use it instead of creating
			// a new one.
			ctlogConfig, err = createConfigFromExistingSecret(ctx, nsSecret, *privateKeySecret)
		} else {
			// Create a fresh private key.
			ctlogConfig, err = createConfigWithKeys(ctx, *keyType)
		}
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to generate keys: %v", err)
		}
		ctlogConfig.PrivKeyPassword = *keyPassword
		ctlogConfig.LogID = treeIDInt
		ctlogConfig.LogPrefix = *ctlogPrefix
		ctlogConfig.TrillianServerAddr = *trillianServerAddr
		if err = ctlogConfig.AddFulcioRoot(ctx, root.ChainPEM); err != nil {
			logging.FromContext(ctx).Infof("Failed to add fulcio root: %v", err)
		}
		configMap, err := ctlogConfig.MarshalConfig(ctx)
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to marshal ctlog config: %v", err)
		}

		if err := secret.ReconcileSecret(ctx, *secretName, ns, configMap, nsSecret); err != nil {
			logging.FromContext(ctx).Fatalf("Failed to reconcile secret: %v", err)
		}

		pubData := map[string][]byte{publicKey: configMap[publicKey]}
		if err := secret.ReconcileSecret(ctx, *pubKeySecretName, ns, pubData, nsSecret); err != nil {
			logging.FromContext(ctx).Panicf("Failed to reconcile public key secret %s/%s: %v", ns, *secretName, err)
		}

		logging.FromContext(ctx).Infof("Created CTLog configuration")
		os.Exit(0)
	}

	// Prefer the secret config if it exists, but if it doesn't use
	// configmap for backwards compatibility / migration.
	if existingSecret.Data[configKey] != nil {
		logging.FromContext(ctx).Infof("Found existing config in the secret, using that %s/%s", ns, *secretName)
	} else {
		existingSecret.Data[configKey] = existingCMConfig
	}

	existingConfig, err := ctlog.Unmarshal(ctx, existingSecret.Data)
	if err != nil {
		log.Fatalf("Failed to unmarshal existing configuration: %v", err)
	}

	// Finally add Fulcio to it, marshal and write out.
	if err = existingConfig.AddFulcioRoot(ctx, root.ChainPEM); err != nil {
		log.Printf("Failed to add fulcio root: %v", err)
	}
	marshaled, err := existingConfig.MarshalConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to marshal new configuration: %v", err)
	}
	// Take out the public / private key from the secret since we didn't mess
	// with those. ReconcileSecret will not touch fields that are not here, so
	// just remove them from the map.
	delete(marshaled, privateKey)
	delete(marshaled, publicKey)
	if err := secret.ReconcileSecret(ctx, *secretName, ns, marshaled, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", ns, *secretName, err)
	}

	pubData := map[string][]byte{publicKey: existingSecret.Data[publicKey]}
	if err := secret.ReconcileSecret(ctx, *pubKeySecretName, ns, pubData, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", ns, *secretName, err)
	}
}

// createConfigWithKeys creates otherwise empty CTLogCOnfig but fills
// in PrivKey, and PubKey. Can not be used as is, but use it to construct
// the base to build upon
func createConfigWithKeys(ctx context.Context, keytype string) (*ctlog.Config, error) {
	var privKey crypto.PrivateKey
	var err error
	if keytype == "rsa" {
		privKey, err = rsa.GenerateKey(rand.Reader, bitSize)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Private RSA Key: %w", err)
		}
	} else {
		privKey, err = ecdsa.GenerateKey(supportedCurves[*curveType], rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Private ECDSA Key: %w", err)
		}
	}

	var ok bool
	var signer crypto.Signer
	if signer, ok = privKey.(crypto.Signer); !ok {
		logging.FromContext(ctx).Fatalf("failed to convert to Signer")
	}
	return &ctlog.Config{
		PrivKey: privKey,
		PubKey:  signer.Public(),
	}, nil
}

// create
func createConfigFromExistingSecret(ctx context.Context, nsSecret v1.SecretInterface, secretName string) (*ctlog.Config, error) {
	existingSecret, err := nsSecret.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting an existing private key secret: %w", err)
	}
	private := existingSecret.Data[privateKey]
	if len(private) == 0 {
		return nil, errors.New("secret missing private key entry")
	}
	priv, pub, err := ctlog.DecryptExistingPrivateKey(private, *keyPassword)
	if err != nil {
		return nil, fmt.Errorf("decrypting existing private key secret: %w", err)
	}
	return &ctlog.Config{
		PrivKey: priv,
		PubKey:  pub,
	}, nil
}
