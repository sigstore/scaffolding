package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	frequency int
	addr      string
	rekorURL  string
	fulcioURL string
)

func init() {
	flag.IntVar(&frequency, "frequecy", 10, "How often to run probers (in seconds)")
	flag.StringVar(&addr, "addr", ":8080", "Port to expose prometheus to")

	flag.StringVar(&rekorURL, "rekor-url", "https://rekor.sigstore.dev", "Set to the Rekor URL to run probers against")
	flag.StringVar(&fulcioURL, "fulcio-url", "https://fulcio.sigstore.dev", "Set to the Fulcio URL to run probers against")

	flag.Parse()
}

func main() {
	reg := prometheus.NewRegistry()
	reg.MustRegister(endpointLatenciesSummary, endpointLatenciesHistogram)

	go runProbers()

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))
	log.Fatal(http.ListenAndServe(addr, nil))
}

func runProbers() {
	for {
		for _, r := range RekorEndpoints {
			if err := observeRequest(rekorURL, r); err != nil {
				fmt.Printf("error running request %s: %v\n", r.endpoint, err)
			}
		}
		for _, r := range FulcioEndpoints {
			if err := observeRequest(fulcioURL, r); err != nil {
				fmt.Printf("error running request %s: %v\n", r.endpoint, err)
			}
		}
		fmt.Println("Complete")
		time.Sleep(time.Duration(frequency) * time.Second)
	}
}

func observeRequest(host string, r ReadProberCheck) error {
	fmt.Println("Observing ", host+r.endpoint)
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

	labels := prometheus.Labels{
		endpointLabel:   r.endpoint,
		statusCodeLabel: fmt.Sprintf("%d", resp.StatusCode),
		hostLabel:       host,
	}
	fmt.Println("Status code: ", resp.StatusCode)
	endpointLatenciesHistogram.With(labels).Observe(float64(latency))
	endpointLatenciesSummary.With(labels).Observe(float64(latency))
	return nil
}

func httpRequest(host string, r ReadProberCheck) (*http.Request, error) {
	req, err := http.NewRequest(r.method, host+r.endpoint, bytes.NewBuffer([]byte(r.body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	q := req.URL.Query()
	for k, v := range r.queries {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	return req, nil
}
