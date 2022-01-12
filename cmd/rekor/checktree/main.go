/*
Copyright 2021 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"log"

	"github.com/go-openapi/strfmt"
	"github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"knative.dev/pkg/signals"
)

var (
	rekorURL = flag.String("rekor_url", "http://rekor.rekor-system.svc", "Address of the Rekor server")
)

func main() {
	flag.Parse()
	if *rekorURL == "" {
		log.Panic("Need a rekorURL")
	}

	ctx := signals.NewContext()
	c, err := client.GetRekorClient(*rekorURL)
	if err != nil {
		log.Panic("Failed to construct rekor client", err)
	}
	entries, err := c.Entries.GetLogEntryByIndex(&entries.GetLogEntryByIndexParams{LogIndex: 0, Context: ctx})
	if err != nil {
		log.Panic("Failed to get entry at index 0", err)
	}
	log.Printf("Got Payload: %+v", entries.Payload)
	if err := entries.Payload.Validate(strfmt.Default); err != nil {
		log.Panic("Failed to validate entry: ", err)
	}
}
