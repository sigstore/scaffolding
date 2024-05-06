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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sigstore/scaffolding/pkg/certs"
	"github.com/sigstore/scaffolding/pkg/repo"
	"github.com/sigstore/scaffolding/pkg/secret"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
	"sigs.k8s.io/release-utils/version"
)

var (
	dir = flag.String("file-dir", "/var/run/tuf-secrets", "Directory where all the files that need to be added to TUF root live. File names are used to as targets.")
	// Name of the "secret" where we create two entries, one for:
	// root = Which holds 1.root.json
	// repository - Compressed repo, which has been tar/gzipped.
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

	tufFiles, err := os.ReadDir(*dir)
	if err != nil {
		logging.FromContext(ctx).Fatalf("failed to read dir %s: %v", *dir, err)
	}
	trimDir := strings.TrimSuffix(*dir, "/")
	files := map[string][]byte{}
	for _, file := range tufFiles {
		if !file.IsDir() {
			logging.FromContext(ctx).Infof("Got file %s", file.Name())
			// Kubernetes adds some extra files here that are prefixed with
			// .., for example '..data' so skip those.
			if strings.HasPrefix(file.Name(), "..") {
				logging.FromContext(ctx).Infof("Skipping .. file %s", file.Name())
				continue
			}
			fileName := fmt.Sprintf("%s/%s", trimDir, file.Name())
			fileBytes, err := os.ReadFile(fileName)
			if err != nil {
				logging.FromContext(ctx).Fatalf("failed to read file %s/%s: %v", fileName, err)
			}
			// If it's a TSA file, we need to split it into multiple TUF
			// targets.
			if strings.Contains(file.Name(), "tsa") {
				logging.FromContext(ctx).Infof("Splitting TSA certchain into individual certs")

				certFiles, err := certs.SplitCertChain(fileBytes, "tsa")
				if err != nil {
					logging.FromContext(ctx).Fatalf("failed to parse  %s/%s: %v", fileName, err)
				}
				for k, v := range certFiles {
					logging.FromContext(ctx).Infof("Got tsa cert file %s", k)
					trimmedCert := strings.TrimSpace(string(v))
					files[k] = []byte(trimmedCert)
				}
			} else {
				files[file.Name()] = fileBytes
			}
		}
	}

	// Create a new TUF root with the listed artifacts.
	local, dir, err := repo.CreateRepo(ctx, files)
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to create repo: %v", err)
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

	// Then compress the root directory and put it into a secret
	// Secrets have 1MiB and the repository as tested goes to about ~3k, so no
	// worries here.
	var compressed bytes.Buffer
	if err := repo.CompressFS(os.DirFS(dir), &compressed, map[string]bool{"keys": true, "staged": true}); err != nil {
		logging.FromContext(ctx).Fatalf("Failed to compress the repo: %v", err)
	}
	data["repository"] = compressed.Bytes()

	nsSecret := clientset.CoreV1().Secrets(ns)
	if err := secret.ReconcileSecret(ctx, *secretName, ns, data, nsSecret); err != nil {
		logging.FromContext(ctx).Panicf("Failed to reconcile secret %s/%s: %v", ns, *secretName, err)
	}
	// Serve the TUF repository.
	logging.FromContext(ctx).Infof("tuf repository was created in: %s", dir)
	serveDir := filepath.Join(dir, "repository")
	logging.FromContext(ctx).Infof("tuf repository was created in: %s serving tuf root at %s", dir, serveDir)
	fs := http.FileServer(http.Dir(serveDir))
	http.Handle("/", fs)

	/* #nosec G114 */
	if err := http.ListenAndServe(":8080", nil); err != nil { //nolint: gosec
		panic(err)
	}
}
