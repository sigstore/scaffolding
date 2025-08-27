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

import "encoding/base64"

var (
	GET  = "GET"
	POST = "POST"
)

type ReadProberCheck struct {
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method"`
	Body        []byte            `json:"body"`
	ContentType string            `json:"contentType"` // if blank and Body != "", defaults to "application/json"
	Accept      string            `json:"accept"`      // if blank, defaults to "application/json"
	Queries     map[string]string `json:"queries"`
	SLOEndpoint string            `json:"slo-endpoint"`
}

// FYI: shard-specific reads are computed in determineShardCoverage
var ShardlessRekorEndpoints = []ReadProberCheck{
	{
		Endpoint: "/api/v1/log/publicKey",
		Method:   GET,
		Accept:   "application/x-pem-file",
	}, {
		Endpoint: "/api/v1/log",
		Method:   GET,
	}, {
		Endpoint: "/api/v1/log/entries/retrieve",
		Method:   POST,
		Body:     []byte(`{"hash":"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b"}`),
	}, {
		Endpoint: "/api/v1/index/retrieve",
		Method:   POST,
		Body:     []byte(`{"hash":"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b"}`),
	},
}

var RekorV2ReadEndpoints = []ReadProberCheck{
	{
		Endpoint: "/healthz",
		Method:   GET,
	},
}

var FulcioEndpoints = []ReadProberCheck{
	{
		Endpoint: "/api/v1/rootCert",
		Method:   GET,
		Accept:   "application/pem-certificate-chain",
	}, {
		Endpoint: "/api/v2/configuration",
		Method:   GET,
	}, {
		Endpoint: "/api/v2/trustBundle",
		Method:   GET,
	},
}

var tsReq, _ = base64.StdEncoding.DecodeString("ME8CAQEwMTANBglghkgBZQMEAgEFAAQg6lDWJ0V9nVEPspa3bDKpG71ef/PswFWOcCjDxLBpe0cCFHsLm2h6a5KYc06qrKCtCIDhwZdmAQH/")
var TSAEndpoints = []ReadProberCheck{
	{
		Endpoint:    "", // TSA endpoints are non-standard and come from the signing config
		Method:      POST,
		Accept:      "application/timestamp-reply",
		ContentType: "application/timestamp-query",
		Body:        tsReq,
	},
}
