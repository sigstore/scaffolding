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
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"

	fulcioclient "github.com/sigstore/fulcio/pkg/api"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

type fulcios []string

func (f *fulcios) String() string {
	return fmt.Sprint(*f)
}

func (f *fulcios) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var fulcioList fulcios

type CertResponse struct {
	Certificates []string `json:"certificates"`
}

func main() {
	flag.Var(&fulcioList, "fulcio", "List of fulcios which must be in the list")
	var ctlogURL = flag.String("ctlog-url", "ctlog.ctlog-system.svc", "CTLog to check Fulcios against.")
	var ctlogPrefix = flag.String("log-prefix", "sigstorescaffolding", "Prefix to append to the gtlogURL url. This is basically the name of the log.")
	flag.Parse()
	var strictMatch = flag.Bool("strict", true, "If set to true ctlog must only contain the Fulcios in the list, no more, no less")
	ctx := signals.NewContext()
	fulcioURLs := make([]*url.URL, 0, len(fulcioList))
	for _, f := range fulcioList {
		u, err := url.Parse(f)
		if err != nil {
			logging.FromContext(ctx).Fatalf("Invalid fulcioURL %s : %v", f, err)
		}
		fulcioURLs = append(fulcioURLs, u)
	}
	fmt.Printf("GOT: %s\n", fulcioList.String())

	// First grab the certs that CTLog has.
	ctlog := fmt.Sprintf("%s/%s/ct/v1/get-roots", *ctlogURL, *ctlogPrefix)
	/* #nosec G107 */
	ctlogResponse, err := http.Get(ctlog)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to get trusted certs from ctlog: %v", err)
	}
	defer ctlogResponse.Body.Close()
	body, err := io.ReadAll(ctlogResponse.Body)
	if err != nil {
		logging.FromContext(ctx).Fatalf("Failed to read body from ctlog: %v", err)
	}
	certs := CertResponse{}
	if err := json.Unmarshal(body, &certs); err != nil {
		logging.FromContext(ctx).Fatalf("Failed to unmarshal body from ctlog: %v", err)
	}
	for i, cert := range certs.Certificates {
		logging.FromContext(ctx).Infof("Got back cert %d: %s", i, cert)
	}

	// Keep track of certs found. Same index as fulcioURLs
	certsFound := make([]bool, len(fulcioURLs))
	for i, fulcio := range fulcioURLs {
		logging.FromContext(ctx).Infof("Fetching fulcio cert %s", fulcio)
		client := fulcioclient.NewClient(fulcio)
		root, err := client.RootCert()
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to fetch fulcio Root %s cert: %v", fulcio.String(), err)
		}
		fulcioCerts, err := cryptoutils.UnmarshalCertificatesFromPEM(root.ChainPEM)
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to unmarshal fulcio Root cert %s cert: %v", fulcio.String(), err)
		}
		fulcioRoot, err := cryptoutils.MarshalCertificateToPEM(fulcioCerts[len(fulcioCerts)-1])
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to marshal fulcio Root cert %s cert: %w", fulcio.String(), err)
		}
		logging.FromContext(ctx).Infof("Got a root cert for fulcio Root cert %s", fulcio)
		// Strip the certificate begin/end marker strings since CTLog doesn't
		// have those.
		block, _ := pem.Decode(fulcioRoot)
		if err != nil {
			logging.FromContext(ctx).Fatalf("Failed to decode fulcio Root PEM %s cert: %w", fulcio.String(), err)
		}
		fulcioRootPEM := []byte(base64.StdEncoding.EncodeToString(block.Bytes))
		for j := range certs.Certificates {
			logging.FromContext(ctx).Infof("Checking ctlog root cert %s", certs.Certificates[j])
			if bytes.Equal(fulcioRootPEM, []byte(certs.Certificates[j])) {
				logging.FromContext(ctx).Infof("Found a matching root cert for fulcio Root cert %s cert: %w", fulcio.String(), err)
				certsFound[i] = true
			}
		}
	}
	// Check that all the expected roots were found.
	allFound := true
	for i := range certsFound {
		if !certsFound[i] {
			logging.FromContext(ctx).Errorf("Did not find a cert for %s", fulcioURLs[i])
			allFound = false
		}
	}
	if !allFound {
		logging.FromContext(ctx).Fatal("Did not find all expected certs")
	}
	// If strict is on, make sure there are no more than expected.
	if *strictMatch && len(certs.Certificates) != len(fulcioURLs) {
		logging.FromContext(ctx).Fatalf("strict mode is on, and CTLog has %d entries and we wanted %d Fulcio entries", len(certs.Certificates), len(certsFound))
	}
}
