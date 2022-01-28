/*
Copyright 2021 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"github.com/go-openapi/runtime"

	"github.com/go-openapi/strfmt"
	"github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/client/index"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/types"
	"github.com/sigstore/rekor/pkg/types/hashedrekord"
	hrv001 "github.com/sigstore/rekor/pkg/types/hashedrekord/v0.0.1"
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

	payload := entries.GetPayload()
	log.Printf("Got Payload: %+v", payload)
	if len(payload) != 1 {
		log.Panic("Payload map length is not 1")
	}

	if err := payload.Validate(strfmt.Default); err != nil {
		log.Panic("Failed to validate entry: ", err)
	}
	for uuid, v := range payload {
		log.Printf("Found UUID: %s", uuid)
		// This has the desired side-effect that it loads the support for
		// unmarshaling below when we call types.NewEntry
		log.Printf("Checking for type: %s version %s", hashedrekord.KIND, hrv001.APIVERSION)
		body, ok := v.Body.(string)
		if !ok {
			log.Panic("Couldn't convert body to string")
		}
		decBody, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			log.Panic("Failed to base64 decode body", err)
		}
		pe, err := models.UnmarshalProposedEntry(bytes.NewReader(decBody), runtime.JSONConsumer())
		if err != nil {
			log.Panic("Failed to unmarshal proposed entry", err)
		}
		hr, err := types.NewEntry(pe)
		if err != nil {
			log.Panic("Failed to convert rekord to known type", err)
		}
		log.Printf("Got TYPE: %+v", hr)
		typed, ok := hr.(*hrv001.V001Entry)
		if !ok {
			log.Panic("Failed to convert rekord to hashrekord", err)
		}
		if typed.HashedRekordObj.Data == nil {
			log.Panic("No data found in hashrekord")
		}
		if typed.HashedRekordObj.Data.Hash == nil {
			log.Panic("No hash found in hashrekord.Data")
		}
		if typed.HashedRekordObj.Data.Hash.Algorithm == nil {
			log.Panic("No hash found in hashrekord.Data.Algorithm")
		}
		if typed.HashedRekordObj.Data.Hash.Value == nil {
			log.Panic("No hash found in hashrekord.Data.Value")
		}

		sha := fmt.Sprintf("%s:%s", *typed.HashedRekordObj.Data.Hash.Algorithm, *typed.HashedRekordObj.Data.Hash.Value)
		log.Printf("Searching for %s", sha)

		// Now that we found the hash, do a query and make sure we get the
		// entry.
		indices, err := c.Index.SearchIndex(index.NewSearchIndexParams().WithQuery(&models.SearchIndex{Hash: sha}))
		if err != nil {
			log.Panic("Failed to query the index: ", err)
		}
		for _, i := range indices.Payload {
			log.Printf("Found index entry: %s", i)
		}
		if len(indices.Payload) != 1 {
			log.Panic("Did not get one entry back from querying the index")
		}
		if indices.Payload[0] != uuid {
			log.Printf("UUIDs do not match, entry %s search returned %s", uuid, indices.Payload[0])
			log.Panic("Did not get expected uuid back from querying the index")
		}
	}
}
