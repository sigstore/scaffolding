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

package repo

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/theupdateframework/go-tuf"
	"knative.dev/pkg/logging"
)

// CreateRepo creates and initializes a Tuf repo for Sigstore by adding
// Fulcio Root Certificate, Rekor, and CTLog public keys into it.
func CreateRepo(ctx context.Context, fulcio, rekor, ctlog []byte) (tuf.LocalStore, string, error) {
	// TODO: Make this an in-memory fileystem.
	tmpDir := os.TempDir()
	dir := tmpDir + "tuf"
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		logging.FromContext(ctx).Errorf("Failed to create tuf dir %v", err)
		return nil, "", err
	}
	dir = dir + "/"
	logging.FromContext(ctx).Infof("Creating the FS in %q", dir)
	local := tuf.FileSystemStore(dir, nil)

	// Create and commit a new TUF repo with the targets to the store.
	logging.FromContext(ctx).Infof("Creating new repo in %q", dir)
	r, err := tuf.NewRepoIndent(local, "", " ")
	if err != nil {
		logging.FromContext(ctx).Errorf("Failed to create NewRepo %v", err)
		return nil, "", err
	}

	// Added by vaikas
	if err := r.Init(false); err != nil {
		logging.FromContext(ctx).Errorf("Failed to init repo %v", err)
		return nil, "", err
	}

	// Make all metadata files expire in 6 months.
	expires := time.Now().AddDate(0, 6, 0)

	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		_, err := r.GenKeyWithExpires(role, expires)
		if err != nil {
			logging.FromContext(ctx).Errorf("Failed to GenKeyWithExpires %v", err)
			return nil, "", err
		}
	}

	// This is the map of targets to add to the trust root with their custom metadata.
	// Use hard-coded names that are used as fallback targets by sigstore's TUF client.
	// TODO(asraa): Update to adding targets in usage subdirectories when sigstore/sigstore#562 is fixed.
	if err := writeStagedTarget(dir, "rekor.pub", []byte(rekor)); err != nil {
		logging.FromContext(ctx).Errorf("Failed to writeStagedTarget for rekor %v", err)
		return nil, "", err
	}
	if err := writeStagedTarget(dir, "fulcio_v1.crt.pem", []byte(fulcio)); err != nil {
		logging.FromContext(ctx).Errorf("Failed to writeStagedTarget for fulcio %v", err)
		return nil, "", err
	}
	if err := writeStagedTarget(dir, "ctfe.pub", []byte(ctlog)); err != nil {
		logging.FromContext(ctx).Errorf("Failed to writeStagedTarget for ctlog %v", err)
		return nil, "", err
	}

	// Now add targets to the TUF repository.
	targets := []string{
		"fulcio_v1.crt.pem",
		"ctfe.pub",
		"rekor.pub",
	}
	err = r.AddTargetsWithExpires(targets, nil, expires)
	if err != nil {
		logging.FromContext(ctx).Errorf("Failed to AddTargets: %v", err)
		return nil, "", err
	}

	// Snapshot, Timestamp, and Publish the repository.
	if err := r.SnapshotWithExpires(expires); err != nil {
		logging.FromContext(ctx).Errorf("Failed to SnashotWithExpires %v", err)
		return nil, "", err
	}
	if err := r.TimestampWithExpires(expires); err != nil {
		logging.FromContext(ctx).Errorf("Failed to TimestampWithExpires %v", err)
		return nil, "", err
	}
	if err := r.Commit(); err != nil {
		logging.FromContext(ctx).Errorf("Failed to Commit %v", err)
		return nil, "", err
	}

	return local, dir, nil
}

func writeStagedTarget(dir, path string, data []byte) error {
	path = filepath.Join(dir, "staged", "targets", path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, []byte(data), 0644); err != nil {
		return err
	}
	return nil
}
