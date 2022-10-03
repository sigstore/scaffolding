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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"sigs.k8s.io/release-utils/version"

	_ "github.com/sigstore/cosign/pkg/providers/all"
)

var (
	frequency      int
	addr           string
	rekorURL       string
	fulcioURL      string
	oneTime        bool
	runWriteProber bool
	versionInfo    version.Info
)

func init() {
	flag.IntVar(&frequency, "frequency", 10, "How often to run probers (in seconds)")
	flag.StringVar(&addr, "addr", ":8080", "Port to expose prometheus to")

	flag.StringVar(&rekorURL, "rekor-url", "https://rekor.sigstore.dev", "Set to the Rekor URL to run probers against")
	flag.StringVar(&fulcioURL, "fulcio-url", "https://fulcio.sigstore.dev", "Set to the Fulcio URL to run probers against")

	flag.BoolVar(&oneTime, "one-time", false, "Whether to run only one time and exit.")
	flag.BoolVar(&runWriteProber, "write-prober", false, " [Kubernetes only] run the probers for the write endpoints.")

	var rekorRequestsJson string
	flag.StringVar(&rekorRequestsJson, "rekor-requests", "[]", "Additional rekor requests (JSON array.)")

	var fulcioRequestsJson string
	flag.StringVar(&fulcioRequestsJson, "fulcio-requests", "[]", "Additional fulcio requests (JSON array.)")

	flag.Parse()

	var rekorFlagRequests []ReadProberCheck
	if err := json.Unmarshal([]byte(rekorRequestsJson), &rekorFlagRequests); err != nil {
		log.Fatal("Failed to parse rekor-requests: ", err)
	}

	var fulcioFlagRequests []ReadProberCheck
	if err := json.Unmarshal([]byte(fulcioRequestsJson), &fulcioFlagRequests); err != nil {
		log.Fatal("Failed to parse fulcio-requests: ", err)
	}

	RekorEndpoints = append(RekorEndpoints, rekorFlagRequests...)
	FulcioEndpoints = append(FulcioEndpoints, fulcioFlagRequests...)
}

func main() {
	ctx := context.Background()
	versionInfo = version.GetVersionInfo()
	fmt.Printf("running create_ct_config Version: %s GitCommit: %s BuildDate: %s", versionInfo.GitVersion, versionInfo.GitCommit, versionInfo.BuildDate)

	reg := prometheus.NewRegistry()
	reg.MustRegister(endpointLatenciesSummary, endpointLatenciesHistogram, verificationCounter)
	reg.MustRegister(NewVersionCollector("sigstore_prober"))

	go runProbers(ctx, frequency, oneTime)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	fmt.Printf("Starting Server on port %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func runProbers(ctx context.Context, freq int, runOnce bool) {
	for {
		hasErr := false

		for _, r := range RekorEndpoints {
			if err := observeRequest(rekorURL, r); err != nil {
				hasErr = true
				fmt.Printf("error running request %s: %v\n", r.Endpoint, err)
			}
		}
		for _, r := range FulcioEndpoints {
			if err := observeRequest(fulcioURL, r); err != nil {
				hasErr = true
				fmt.Printf("error running request %s: %v\n", r.Endpoint, err)
			}
		}
		if runWriteProber {
			if err := fulcioWriteEndpoint(ctx); err != nil {
				hasErr = true
				fmt.Printf("error running fulcio write prober: %v\n", err)
			}
			if err := rekorWriteEndpoint(ctx); err != nil {
				hasErr = true
				fmt.Printf("error running rekor write prober: %v\n", err)
			}
		}

		if runOnce {
			if hasErr {
				fmt.Println("Failed")
				os.Exit(1)
			} else {
				fmt.Println("Complete")
				os.Exit(0)
			}
		}

		time.Sleep(time.Duration(frequency) * time.Second)
	}
}

func observeRequest(host string, r ReadProberCheck) error {
	client := &http.Client{}

	req, err := httpRequest(host, r)
	if err != nil {
		return err
	}

	s := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(s).Milliseconds()

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	exportDataToPrometheus(resp, host, r.Endpoint, r.Method, latency)
	return nil
}

func httpRequest(host string, r ReadProberCheck) (*http.Request, error) {
	req, err := http.NewRequest(r.Method, host+r.Endpoint, bytes.NewBuffer([]byte(r.Body)))
	if err != nil {
		return nil, err
	}

	setHeaders(req, "")
	q := req.URL.Query()
	for k, v := range r.Queries {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}
