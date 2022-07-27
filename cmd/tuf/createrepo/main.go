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
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/theupdateframework/go-tuf"
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
	rekorURL         = flag.String("rekor_url", "http://rekor.rekor-system.svc", "Address of the Rekor server")
	fulcioCert       = flag.String("fulcio-cert", "", "Path to the fulcio certificate")
	ctPubKey         = flag.String("ct-pubkey", "", "Path to a CT Log public key")
	secretName       = flag.String("secret", "tuf-secrets", "Name of the secret to create for the initial root file")
	pubKeySecretName = flag.String("rootsecret", "tuf-root", "Name of the secret to create containing only the initial root")
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

	// Create a new TUF root with the listed artifacts.

	// Add the initial root to secrets.
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

	// Serve the TUF repo
	fs := http.FileServer(http.Dir("./repo"))
	http.Handle("/", fs)

	if err := http.ListenAndServe(":8000", nil); err != nil {
		panic(err)
	}
}

func createRepo(local tuf.LocalStore, targets map[string]json.RawMessage) error {
	local := tuf.FileSystemStore(dir)

	// Create and commit a new TUF repo with the targets to the store.
	r, err := tuf.NewRepo(local)
	if err != nil {
		return err
	}

	// Make all metadata files expire in 6 months.
	expires := time.Now().AddDate(0, 6, 0)

	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		_, err := r.GenKeyWithExpires(role, expires)
		if err != nil {
			return err
		}
	}

	for targetName, customMetadata := range targets {
		r.AddTargetsWithExpires([]string{targetName}, customMetadata, expires)
	}

	if err := r.SnapshotWithExpires(expires); err != nil {
		return err
	}
	if err := r.TimestampWithExpires(expires); err != nil {
		return err
	}
	if err := r.Commit(); err != nil {
		return err
	}

	return nil
}

func (t *tmpDir) writeStagedTarget(path, data string) error {
	path = filepath.Join(t.path, "staged", "targets", path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, []byte(data), 0644); err != nil {
		return err
	}
	return nil
}

func createMetadata(usage string) json.RawMessage {
	return nil
}
