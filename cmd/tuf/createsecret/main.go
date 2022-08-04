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
	"flag"
	"os"

	"github.com/sigstore/scaffolding/pkg/secret"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

const (
	// These are the keys held in the secrets created by the ctlog/certs and
	// fulcio/certs jobs.
	fulcioSecretKey = "cert"
	ctSecretKey     = "public"
	rekorSecretKey  = "public"

	// These are the fields we create in the secret specified with the
	// --secret-name flag.
	fulcioSecretKeyOut = "fulcio-cert"
	ctSecretKeyOut     = "ctlog-pubkey"
	rekorSecretKeyOut  = "rekor-pubkey"
)

var (
	fulcioSecret = flag.String("fulcio-secret", "fulcio-pub-key", "Secret holding Fulcio cert")
	// URL to Rekor to query for the Rekor public key to include in the trust root.
	// Could replace with rekor-pubkey, if that can be resolved early enough (don't know)
	rekorSecret = flag.String("rekor-secret", "rekor-pub-key", "Secret holding Rekor public key")
	ctlogSecret = flag.String("ctlog-secret", "ctlog-public-key", "Secret holding CTLog public key")
	secretName  = flag.String("secret-name", "tuf-secrets", "Name of the secret to create holding necessary information to create/run tuf service")
)

func main() {
	flag.Parse()
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}
	ctx := signals.NewContext()

	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running create_secrets Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get clientset: %v", err)
	}

	// Grab the Fulcio root cert
	nsSecret := clientset.CoreV1().Secrets(ns)
	fs, err := nsSecret.Get(ctx, *fulcioSecret, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *fulcioSecret, err)
	}
	fCert := fs.Data[fulcioSecretKey]
	if fCert == nil || len(fCert) == 0 {
		logging.FromContext(ctx).Panicf("Fulcio cert key %q is missing %s/%s", fulcioSecretKey, ns, *fulcioSecret)
	}
	// Grab the ctlog public key
	cs, err := nsSecret.Get(ctx, *ctlogSecret, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *ctlogSecret, err)
	}
	cPub := cs.Data[ctSecretKey]
	if cPub == nil || len(cPub) == 0 {
		logging.FromContext(ctx).Panicf("Ctlog public key %q is missing %s/%s", ctSecretKey, ns, *ctlogSecret)
	}
	// Grab the rekor public key
	rs, err := nsSecret.Get(ctx, *rekorSecret, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *rekorSecret, err)
	}
	rPub := rs.Data[rekorSecretKey]
	if cPub == nil || len(cPub) == 0 {
		logging.FromContext(ctx).Panicf("Ctlog public key %q is missing %s/%s", ctSecretKey, ns, *ctlogSecret)
	}

	data := map[string][]byte{
		fulcioSecretKeyOut: fCert,
		ctSecretKeyOut:     cPub,
		rekorSecretKeyOut:  rPub,
	}

	if err := secret.ReconcileSecret(ctx, *secretName, ns, data, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *secretName, err)
	}
}
