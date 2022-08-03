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
	"flag"
	"os"

	"github.com/sigstore/cosign/cmd/cosign/cli/rekor"
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
	// These are the keys held in the secrets created by the ctlog/certs and
	// fulcio/certs jobs.
	fulcioSecretKey = "cert"
	ctSecretKey     = "public"

	// These are the fields we create in the secret specified with the
	// --secret-name flag.
	fulcioSecretKeyOut = "fulcio-cert"
	ctSecretKeyOut     = "ctlog-pubkey"
	rekorSecretKeyOut  = "rekor-pubkey"
)

var (
	fulcioSecret = flag.String("fulcio-secret", "fulcio-secret", "Secret holding Fulcio cert")
	// URL to Rekor to query for the Rekor public key to include in the trust root.
	// Could replace with rekor-pubkey, if that can be resolved early enough (don't know)
	rekorURL = flag.String("rekor-url", "http://rekor.rekor-system.svc", "Address of the Rekor server")
	// TODO(vaikas): Consider creating this secret possibly.
	/*
		rekorSecret   = flag.String("rekor-secret", "fulcio-secrets", "Secret holding Fulcio cert")
	*/
	ctlogSecret = flag.String("ctlog-secret", "ctlog-public-key", "Secret holding CTLog public key")

	secretName = flag.String("secret-name", "tuf-secrets", "Name of the secret to create holding necessary information to create/run tuf service")
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
	nsSecrets := clientset.CoreV1().Secrets(ns)
	fs, err := nsSecrets.Get(ctx, *fulcioSecret, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *fulcioSecret, err)
	}
	fCert := fs.Data[fulcioSecretKey]
	if fCert == nil || len(fCert) == 0 {
		logging.FromContext(ctx).Panicf("Fulcio cert key %q is missing %s/%s", fulcioSecretKey, ns, *fulcioSecret)
	}
	// Grab the  ctlog public key
	cs, err := nsSecrets.Get(ctx, *ctlogSecret, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *ctlogSecret, err)
	}
	cPub := cs.Data[ctSecretKey]
	if cPub == nil || len(cPub) == 0 {
		logging.FromContext(ctx).Panicf("Ctlog public key %q is missing %s/%s", ctSecretKey, ns, *ctlogSecret)
	}

	// And finally grab the Rekor data from the server.
	rekorClient, err := rekor.NewClient(*rekorURL)
	if err != nil {
		logging.FromContext(ctx).Panicf("Unable to get rekor client: %v", err)
	}
	rekorKey, err := rekorClient.Pubkey.GetPublicKey(nil)
	if err != nil {
		logging.FromContext(ctx).Panicf("Unable to fetch rkeor key: %v", err)
	}

	data := make(map[string][]byte)
	data[fulcioSecretKeyOut] = fCert
	data[ctSecretKeyOut] = cPub
	data[rekorSecretKeyOut] = []byte(rekorKey.Payload)

	// See if there's an existing secret first
	existingSecret, err := nsSecrets.Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *secretName, err)
	}

	// If we found the secret, just make sure all the fields are there.
	if err == nil && existingSecret != nil && existingSecret.Data != nil {
		esd := existingSecret.Data
		update := false
		if bytes.Compare(esd[fulcioSecretKeyOut], data[fulcioSecretKeyOut]) != 0 {
			logging.FromContext(ctx).Infof("Fulcio key missing or different than expected, updating")
			update = true
		}
		if bytes.Compare(esd[ctSecretKeyOut], data[ctSecretKeyOut]) != 0 {
			logging.FromContext(ctx).Infof("CTLog key missing or different than expected, updating")
		}
		if bytes.Compare(esd[rekorSecretKeyOut], data[rekorSecretKeyOut]) != 0 {
			logging.FromContext(ctx).Infof("Rekor key missing or different than expected, updating")
		}
		if !update {
			logging.FromContext(ctx).Infof("Found existing secret config with all the expected keys")
			return
		}
		existingSecret.Data = data
		_, err = nsSecrets.Update(ctx, existingSecret, metav1.UpdateOptions{})
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to update secret %s/%s: %v", ns, *secretName, err)
		}
		logging.FromContext(ctx).Infof("Updated secret %s/%s", ns, *secretName)
		return
	}

	// Secret is not there, so just create id.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      *secretName,
		},
		Data: data,
	}
	_, err = nsSecrets.Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to create secret %s/%s: %v", ns, *secretName, err)
	}
	logging.FromContext(ctx).Infof("Created secret %s/%s", ns, *secretName)
}
