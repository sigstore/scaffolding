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

// rekorV2ReadURLsFromTUF fetches the URLs for V2 from the signing-config.json.
// Only services with a validity date end after now will be included.
func rekorV2ReadURLsFromTUF(tufMirror string) ([]string, error) {
	rekorV2ServiceConfigs, err := rekorV2ServiceConfigsFromTUF(tufMirror)
	if err != nil {
		return nil, err
	}
	uRLs := []string{}
	for _, s := range rekorV2ServiceConfigs {
		// Read, regardless of startDate, but respect the endDate.
		if s.MajorAPIVersion == 2 && s.ValidityPeriodEnd.After(time.Now()) {
			uRLs = append(uRLs, s.URL)
		}
	}
	Logger.Debug(fmt.Sprintf("fetched rekorV2 read URLs from TUF: %v", uRLs))
	return uRLs, nil
}

// rekorV2WriteURLsFromTUF fetches the write URLs for V2 from the signing-config.json.
// Only services with where we are currently within the validity start and end dates will be included.
func rekorV2WriteURLsFromTUF(tufMirror string) ([]string, error) {
	rekorV2ServiceConfigs, err := rekorV2ServiceConfigsFromTUF(tufMirror)
	if err != nil {
		return nil, err
	}
	uRLs := []string{}
	for _, s := range rekorV2ServiceConfigs {
		// write, only if within the validity period.
		if s.MajorAPIVersion == 2 && s.ValidityPeriodStart.Before(time.Now()) && s.ValidityPeriodEnd.After(time.Now()) {
			uRLs = append(uRLs, s.URL)
		}
	}
	Logger.Debug(fmt.Sprintf("fetched rekorV2 write URLs from TUF: %v", uRLs))
	return uRLs, nil
}

// rekorV2ServiceConfigsFromTUF fetches the Service configs for RekorV2 from the signing-config.json.
func rekorV2ServiceConfigsFromTUF(tufMirror string) ([]*root.Service, error) {
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
	rekorV2ServiceConfigs := []*root.Service{}
	for _, s := range signingConfig.RekorLogURLs() {
		if s.MajorAPIVersion == 2 {
			rekorV2ServiceConfigs = append(rekorV2ServiceConfigs, &s)
		}
	}
	return rekorV2ServiceConfigs, nil
}
