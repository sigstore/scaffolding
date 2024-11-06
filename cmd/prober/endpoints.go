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

var (
	GET  = "GET"
	POST = "POST"
)

type ReadProberCheck struct {
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method"`
	Body        string            `json:"body"`
	Queries     map[string]string `json:"queries"`
	SLOEndpoint string            `json:"slo-endpoint"`
}

// FYI: shard-specific reads are computed in determineShardCoverage
var RekorEndpoints = []ReadProberCheck{
	{
		Endpoint: "/api/v1/log/publicKey",
		Method:   GET,
	}, {
		Endpoint: "/api/v1/log",
		Method:   GET,
	}, {
		Endpoint: "/api/v1/log/proof",
		Method:   GET,
		Queries:  map[string]string{"firstSize": "10", "lastSize": "20"},
	}, {
		Endpoint: "/api/v1/log/entries/retrieve",
		Method:   POST,
		Body:     "{\"hash\":\"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b\"}",
	}, {
		Endpoint: "/api/v1/index/retrieve",
		Method:   POST,
		Body:     "{\"hash\":\"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b\"}",
	},
}

var FulcioEndpoints = []ReadProberCheck{
	{
		Endpoint: "/api/v1/rootCert",
		Method:   GET,
	}, {
		Endpoint: "/api/v2/configuration",
		Method:   GET,
	}, {
		Endpoint: "/api/v2/trustBundle",
		Method:   GET,
	},
}
