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

//go:build tools
// +build tools

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
	_ "github.com/transparency-dev/tesseract/cmd/tesseract/gcp"
	_ "github.com/transparency-dev/tesseract/cmd/tesseract/posix"
	_ "github.com/transparency-dev/witness/cmd/gcp/omniwitness"
)
