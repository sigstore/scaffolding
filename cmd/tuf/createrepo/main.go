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
	"io/ioutil"
	"net/http"
	"os"

	"github.com/sigstore/scaffolding/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

var (
	// Static data to include in the trust root.
	rekorPubKey = flag.String("rekor-pubkey", "/var/run/tuf-secrets/rekor-pubkey", "Path to public key of Rekor server")
	fulcioCert  = flag.String("fulcio-cert", "/var/run/tuf-secrets/fulcio-cert", "Path to the fulcio certificate")
	ctPubKey    = flag.String("ct-pubkey", "/var/run/tuf-secrets/ct-pubkey", "Path to a CT Log public key")
	// Name of the "secret" initial 1.root.json.
	secretName = flag.String("rootsecret", "tuf-root", "Name of the secret to create for the initial root file")
)

func main() {
	flag.Parse()

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}
	ctx := signals.NewContext()

	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running create_repo Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to get clientset: %v", err)
	}

	// Read the Rekor file
	rekor, err := ioutil.ReadFile(*rekorPubKey)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to read Rekor pubkey %s: %v", *rekorPubKey, err)
	}

	fulcio, err := ioutil.ReadFile(*fulcioCert)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to read Fulcio cert %s: %v", *fulcioCert, err)
	}

	ct, err := ioutil.ReadFile(*ctPubKey)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to read ctPubkey %s: %v", *ctPubKey, err)
	}

	// Create a new TUF root with the listed artifacts.
	local, dir, err := repo.CreateRepo(ctx, fulcio, rekor, ct)
	if err != nil {
		logging.FromContext(ctx).Panicf("Creating repo: %v", err)
	}
	meta, err := local.GetMeta()
	if err != nil {
		logging.FromContext(ctx).Panicf("Getting meta: %v", err)
	}
	rootJSON, ok := meta["root.json"]
	if !ok {
		logging.FromContext(ctx).Panicf("Getting root: %v", err)
	}

	// Add the initial 1.root.json to secrets.
	data := make(map[string][]byte)
	data["root"] = rootJSON

	existingSecret, err := clientset.CoreV1().Secrets(ns).Get(ctx, *secretName, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		logging.FromContext(ctx).Panicf("Failed to get secret %s/%s: %v", ns, *secretName, err)
	}

	if err == nil && existingSecret != nil {
		_, rootok := existingSecret.Data["root"]

		if rootok {
			logging.FromContext(ctx).Infof("Found existing secret config with the TUF root")
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

	// Serve the TUF repository.
	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", fs)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
