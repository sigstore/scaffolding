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
	"log"
	"os"

	"github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/scaffolding/pkg/secret"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

var (
	rekorURL   = flag.String("rekor_url", "http://rekor.rekor-system.svc", "Address of the Rekor server")
	secretName = flag.String("secret", "rekor-public-key", "Secret to create the rekor public key in.")
)

func main() {
	flag.Parse()
	if *rekorURL == "" {
		log.Panic("Need a rekorURL")
	}
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		panic("env variable NAMESPACE must be set")
	}

	ctx := signals.NewContext()
	versionInfo := version.GetVersionInfo()
	logging.FromContext(ctx).Infof("running createsecret Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	config, err := rest.InClusterConfig()
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get InClusterConfig: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get clientset: %v", err)
	}
	nsSecrets := clientset.CoreV1().Secrets(ns)

	c, err := client.GetRekorClient(*rekorURL)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to construct rekor client", err)
	}
	// Create a secret holding our public key for OOB consumption of it.
	// I know it's little bit fake because we call into it but at least we
	// create it here so at least it's available for OOB.
	rekorKey, err := c.Pubkey.GetPublicKey(nil)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Unable to fetch rekor key: %v", err)
	}

	data := map[string][]byte{"public": []byte(rekorKey.Payload)}
	if err := secret.ReconcileSecret(ctx, *secretName, ns, data, nsSecrets); err != nil {
		logging.FromContext(ctx).Fatalf("Unable to reconcile secret: %v", err)
	}
}
