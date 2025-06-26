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
)

// rekorV2ServiceURLsFromTUF fetches the URLs for V2 from the signing-config.json.
// Only services with a validityStart before now will be included.
func rekorV2ServiceURLsFromTUF(tufMirror string) ([]string, error) {
	opts := tuf.DefaultOptions().WithRepositoryBaseURL(tufMirror).WithRoot([]byte(tufRootJSON))
	client, err := tuf.New(opts)
	if err != nil {
		return nil, err
	}
	// TODO: use root.GetSigningConfig(client), when the TUF repo's signing_config.json starts using v0.2.
	signingConfigBytes, err := client.GetTarget("signing_config.v0.2.json")
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
	Logger.Debug(fmt.Sprintf("fetch urls from TUF: %v", uRLs))
	return uRLs, nil
}
