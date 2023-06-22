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
	tsaCertChainKey = "cert-chain"

	// These are the fields we create in the secret specified with the
	// --secret-name flag.
	// These are the target names that are created, so the field names
	// are the names of the files that then get mounted onto the container, so
	// we can then just slurp them in and create TUF root out of.
	/* #nosec G101 */
	fulcioSecretKeyOut = "fulcio_v1.crt.pem"
	/* #nosec G101 */
	ctSecretKeyOut = "ctfe.pub"
	/* #nosec G101 */
	rekorSecretKeyOut = "rekor.pub"
	tsaCertChainOut   = "tsa.certchain.pem"
)

var (
	fulcioSecret = flag.String("fulcio-secret", "fulcio-pub-key", "Secret holding Fulcio cert")
	rekorSecret  = flag.String("rekor-secret", "rekor-pub-key", "Secret holding Rekor public key")
	ctlogSecret  = flag.String("ctlog-secret", "ctlog-public-key", "Secret holding CTLog public key")
	tsaSecret    = flag.String("tsa-secret", "tsa-cert-chain", "Secret holding the TSA certificate chain")
	secretName   = flag.String("secret-name", "tuf-secrets", "Name of the secret to create holding necessary information to create/run tuf service")
)

func main() {
	flag.Parse()
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}
	ctx := signals.NewContext()

	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running createsecret Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

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
	if len(fCert) == 0 {
		logging.FromContext(ctx).Panicf("Fulcio cert key %q is missing %s/%s", fulcioSecretKey, ns, *fulcioSecret)
	}
	// Grab the ctlog public key
	cs, err := nsSecret.Get(ctx, *ctlogSecret, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *ctlogSecret, err)
	}
	cPub := cs.Data[ctSecretKey]
	if len(cPub) == 0 {
		logging.FromContext(ctx).Panicf("Ctlog public key %q is missing %s/%s", ctSecretKey, ns, *ctlogSecret)
	}
	// Grab the rekor public key
	rs, err := nsSecret.Get(ctx, *rekorSecret, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *rekorSecret, err)
	}
	rPub := rs.Data[rekorSecretKey]
	if len(rPub) == 0 {
		logging.FromContext(ctx).Panicf("Rekor public key %q is missing %s/%s", rekorSecretKey, ns, *rekorSecret)
	}

	// Grab the TSA cert chain public key
	tsa, err := nsSecret.Get(ctx, *tsaSecret, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get tsa secret %s/%s: %v", ns, *tsaSecret, err)
	}
	certChain := tsa.Data[tsaCertChainKey]
	if len(certChain) == 0 {
		logging.FromContext(ctx).Panicf("TSA Cert chain key %q is missing %s/%s", tsaCertChainKey, ns, *tsaSecret)
	}

	data := map[string][]byte{
		fulcioSecretKeyOut: fCert,
		ctSecretKeyOut:     cPub,
		rekorSecretKeyOut:  rPub,
		tsaCertChainOut:    certChain,
	}

	if err := secret.ReconcileSecret(ctx, *secretName, ns, data, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *secretName, err)
	}
}
