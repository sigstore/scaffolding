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
	"github.com/sigstore/scaffolding/pkg/secret"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	configInSecret     = flag.Bool("config-in-secret", false, "If set to true, create the ctlog configuration proto into a secret specified in ctlog-secrets under key 'config'")
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

	var marshalledConfig []byte

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
		ExtKeyUsages:   []string{"CodeSigning"},
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
	marshalledConfig, err = prototext.Marshal(&multiConfig)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to marshal config proto: %v", err)
	}

	// If ctlog config is not supposed to be put into a secret, and config
	// doesn't exist, or is out of date, update ConfigMap.
	if !*configInSecret && (cm.BinaryData == nil || bytes.Compare(marshalledConfig, cm.BinaryData[configKey]) != 0) {
		logging.FromContext(ctx).Infof("Updating config with treeid: %s", treeID)
		if cm.BinaryData == nil {
			cm.BinaryData = make(map[string][]byte)
		}
		cm.BinaryData[configKey] = marshalledConfig
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

	// Fetch only root certificate from the chain
	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(root.ChainPEM)
	if err != nil {
		logging.FromContext(ctx).Panicf("unable to unmarshal certficate chain: %v", err)
	}
	rootCertPEM, err := cryptoutils.MarshalCertificateToPEM(certs[len(certs)-1])
	if err != nil {
		logging.FromContext(ctx).Panicf("unable to marshal root certificate: %v", err)
	}

	data := map[string][]byte{
		"private": privPEM,
		"public":  pubPEM,
		"rootca":  rootCertPEM,
	}

	// If the config is supposed to be written into a secret, make it so.
	if *configInSecret {
		data[configKey] = marshalledConfig
	}

	nsSecret := clientset.CoreV1().Secrets(*ns)
	if err := secret.ReconcileSecret(ctx, *secretName, *ns, data, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", *ns, *secretName, err)
	}

	pubData := map[string][]byte{"public": pubPEM}
	if err := secret.ReconcileSecret(ctx, *pubKeySecretName, *ns, pubData, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", *ns, *secretName, err)
	}
}

func mustMarshalAny(pb proto.Message) *anypb.Any {
	ret, err := anypb.New(pb)
	if err != nil {
		panic(fmt.Sprintf("MarshalAny failed: %v", err))
	}
	return ret
}
