//go:build tools
// +build tools

/*
Copyright 2021 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package tools

import (
	_ "k8s.io/code-generator"
	_ "knative.dev/hack"

	// codegen: hack/generate-knative.sh
	_ "knative.dev/pkg/hack"

	// Import things that we build using ko
	_ "github.com/google/certificate-transparency-go/trillian/ctfe/ct_server"
	_ "github.com/google/trillian/cmd/trillian_log_server"
	_ "github.com/google/trillian/cmd/trillian_log_signer"
	_ "github.com/sigstore/rekor/cmd/rekor-server"
)
