package main

import (
	"flag"
	"fmt"
	"io"
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
	go runProbers()

	prometheus.MustRegister(endpointLatencies)

	// Expose the registered metrics via HTTP.
	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
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
			if err := runRequest(rekorURL, r); err != nil {
				fmt.Printf("error running request %s: %v\n", r.endpoint, err)
			}
		}
		time.Sleep(time.Duration(frequency) * time.Second)
	}
}

func runRequest(host string, r ReadProberCheck) error {
	client := &http.Client{}

	s := time.Now()
	req, err := http.NewRequest(r.method, host+r.endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	latency := time.Since(s).Milliseconds()

	endpointLatencies.WithLabelValues(r.endpoint).Observe(float64(latency))

	fmt.Println("latency:", latency)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(bytes))
	}
	return nil
}
