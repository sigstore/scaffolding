// Copyright 2023 The Sigstore Authors
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
	"log"
	"os"
	"os/exec"
	"strings"

	"chainguard.dev/exitdir"
	"knative.dev/pkg/signals"
)

// Assuming the base image is image: gcr.io/cloud-sql-connectors/cloud-sql-proxy

func main() {
	// Leverage exitdir to use file based lifecycle management.
	ctx := exitdir.Aware(signals.NewContext())

	log.Println("Starting the cloud sql proxy...")
	cmd := exec.CommandContext(ctx, "/cloud-sql-proxy", os.Args[1:]...) //nolint: gosec
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil && strings.Contains(err.Error(), "signal: killed") {
		log.Println("Got signal to shutdown")
	} else if err != nil {
		log.Panic(err)
	}

	<-ctx.Done()
	log.Println("Exiting cloud sql proxy...")
}
