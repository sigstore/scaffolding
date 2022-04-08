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
	"encoding/pem"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/google/certificate-transparency-go/trillian/ctfe/configpb"
	"github.com/google/trillian/crypto/keyspb"
	fulcioclient "github.com/sigstore/fulcio/pkg/api"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	treeKey   = "treeID"
	configKey = "config"
	bitSize   = 4096
)

var (
	ns                 = flag.String("namespace", "ctlog-system", "Namespace where to get the configmap containing treeid")
	cmname             = flag.String("configmap", "ctlog-config", "Name of the configmap where the treeID lives")
	secretName         = flag.String("secret", "ctlog-secrets", "Name of the secret to create for the keyfiles")
	pubKeySecretName   = flag.String("pubkeysecret", "ctlog-public-key", "Name of the secret to create containing only the public key")
	ctlogPrefix        = flag.String("log-prefix", "sigstorescaffolding", "Prefix to append to the url. This is basically the name of the log.")
	fulcioURL          = flag.String("fulcio-url", "http://fulcio.fulcio-system.svc", "Where to fetch the fulcio Root CA from")
	trillianServerAddr = flag.String("trillian-server", "log-server.trillian-system.svc:80", "Address of the gRPC Trillian Admin Server (host:port)")
	keyPassword        = flag.String("key-password", "test", "Password for the PEM key")
	pemPassword        = flag.String("pem-password", "test", "Password for encrypting PEM")
)

func main() {
	flag.Parse()

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
	cm, err := clientset.CoreV1().ConfigMaps(*ns).Get(ctx, *cmname, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get the configmap %s/%s: %v", *ns, *cmname, err)
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

	// Generate RSA key. We do it here in case we need to update the config
	// with it.
	key, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		panic(err)
	}

	if _, ok := cm.Data[configKey]; !ok {
		privKeyProto := mustMarshalAny(&keyspb.PEMKeyFile{Path: "/ctfe-keys/privkey.pem", Password: *keyPassword})

		keyDER, err := x509.MarshalPKIXPublicKey(key.Public())
		if err != nil {
			logging.FromContext(ctx).Panicf("Failed to marshal the public key: %v", err)
		}
		proto := configpb.LogConfig{
			LogId:          treeIDInt,
			Prefix:         *ctlogPrefix,
			RootsPemFile:   []string{"/ctfe-keys/roots.pem"},
			PrivateKey:     privKeyProto,
			PublicKey:      &keyspb.PublicKey{Der: keyDER},
			LogBackendName: "trillian",
		}

		multiConfig := configpb.LogMultiConfig{
			LogConfigs: &configpb.LogConfigSet{
				Config: []*configpb.LogConfig{&proto},
			},
			Backends: &configpb.LogBackendSet{
				Backend: []*configpb.LogBackend{{
					Name:        "trillian",
					BackendSpec: *trillianServerAddr,
				}},
			},
		}
		marshalled, err := prototext.Marshal(&multiConfig)
		if err != nil {
			logging.FromContext(ctx).Panicf("Failed to marshal config proto: %v", err)
		}
		logging.FromContext(ctx).Infof("Updating config with treeid: %s", treeID)
		if cm.BinaryData == nil {
			cm.BinaryData = make(map[string][]byte)
		}

		cm.BinaryData[configKey] = marshalled
		_, err = clientset.CoreV1().ConfigMaps(*ns).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			logging.FromContext(ctx).Panicf("Failed to update the configmap %s/%s: %v", *ns, *cmname, err)
		}
	}
	// Extract public component.
	pub := key.Public()

	// Encode private key to PKCS#1 ASN.1 PEM.
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	// Encrypt the pem
	block, err = x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(*pemPassword), x509.PEMCipherAES256) // nolint
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to encrypt private key: %v", err)
	}

	privPEM, err := pem.EncodeToMemory(block), nil
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to encode encrypted private key: %v", err)
	}
	// Encode public key to PKCS#1 ASN.1 PEM.
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
		},
	)

	data := make(map[string][]byte)
	data["private"] = privPEM
	data["public"] = pubPEM
	data["rootca"] = root.ChainPEM

	existingSecret, err := clientset.CoreV1().Secrets(*ns).Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", *ns, *secretName, err)
	}

	if err == nil && existingSecret != nil {
		_, privok := existingSecret.Data["private"]
		_, pubok := existingSecret.Data["public"]

		if privok && pubok {
			logging.FromContext(ctx).Infof("Found existing secret config with keys")
			return
		}
		existingSecret.Data = data
		_, err = clientset.CoreV1().Secrets(*ns).Update(ctx, existingSecret, metav1.UpdateOptions{})
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to update secret %s/%s: %v", *ns, *secretName, err)
		}
		logging.FromContext(ctx).Infof("Updated existing secret config with keys")
	} else {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: *ns,
				Name:      *secretName,
			},
			Data: data,
		}
		_, err = clientset.CoreV1().Secrets(*ns).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to create secret %s/%s: %v", *ns, *secretName, err)
		}
	}

	pubData := make(map[string][]byte)
	pubData["public"] = pubPEM

	existingPubSecret, err := clientset.CoreV1().Secrets(*ns).Get(ctx, *pubKeySecretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", *ns, *pubKeySecretName, err)
	}

	if err == nil && existingPubSecret != nil {
		if _, pubok := existingPubSecret.Data["public"]; pubok {
			logging.FromContext(ctx).Infof("Found existing secret config with public key")
			return
		}
		existingPubSecret.Data = pubData
		_, err = clientset.CoreV1().Secrets(*ns).Update(ctx, existingPubSecret, metav1.UpdateOptions{})
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to update secret %s/%s: %v", *ns, *pubKeySecretName, err)
		}
		logging.FromContext(ctx).Infof("Updated existing secret config with keys")
		return
	}

	pubSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: *ns,
			Name:      *pubKeySecretName,
		},
		Data: pubData,
	}
	_, err = clientset.CoreV1().Secrets(*ns).Create(ctx, pubSecret, metav1.CreateOptions{})
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to create public key secret %s/%s: %v", *ns, *pubKeySecretName, err)
	}
}

func mustMarshalAny(pb proto.Message) *anypb.Any {
	ret, err := anypb.New(pb)
	if err != nil {
		panic(fmt.Sprintf("MarshalAny failed: %v", err))
	}
	return ret
}
