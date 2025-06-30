// Copyright 2025 The Sigstore Authors
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
	"fmt"
	"time"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore/pkg/signature"
)

func getTufClient(tufMirror string, tufRootJSON string) (*tuf.Client, error) {
	opts := tuf.DefaultOptions().WithRepositoryBaseURL(tufMirror).WithRoot([]byte(tufRootJSON))
	return tuf.New(opts)
}

// rekorV2ServiceURLsFromTUF fetches the URLs for V2 from the signing-config.json.
// Only services with a validityStart before now will be included.
func rekorV2ServiceURLsFromTUF(tufClient *tuf.Client) ([]string, error) {
	// TODO: use root.GetSigningConfig(client), when the TUF repo's signing_config.json starts using v0.2.
	signingConfigBytes, err := tufClient.GetTarget("signing_config.v0.2.json")
	if err != nil {
		return nil, err
	}
	signingConfig, err := root.NewSigningConfigFromJSON(signingConfigBytes)
	if err != nil {
		return nil, err
	}
	uRLs := []string{}
	for _, s := range signingConfig.RekorLogURLs() {
		if s.MajorAPIVersion == 2 && s.ValidityPeriodStart.Before(time.Now()) {
			uRLs = append(uRLs, s.URL)
		}
	}
	Logger.Debug(fmt.Sprintf("fetched rekorV2 urls from TUF: %v", uRLs))
	return uRLs, nil
}

func rekorV2VerifierFromTUF(tufClient *tuf.Client, rekorURL string) (*signature.Verifier, error) {
	trustedRoot, err := root.GetTrustedRoot(tufClient)
	if err != nil {
		return nil, err
	}
	var transparencyLog *root.TransparencyLog
	for _, log := range trustedRoot.RekorLogs() {
		if log.BaseURL == rekorURL {
			transparencyLog = log
		}
	}
	if transparencyLog == nil {
		return nil, fmt.Errorf("rekorV2 public key not found in TUF for rekorURL: %s", rekorURL)
	}
	verifier, err := signature.LoadVerifier(transparencyLog.PublicKey, transparencyLog.HashFunc)
	if err != nil {
		return nil, fmt.Errorf("loading verifier: %w", err)
	}
	return &verifier, nil
}
