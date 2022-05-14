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
	endpoint string
	method   string
	body     string
	queries  map[string]string
}

var RekorEndpoints = []ReadProberCheck{
	{
		endpoint: "/api/v1/version",
		method:   GET,
	}, {
		endpoint: "/api/v1/log/publicKey",
		method:   GET,
	},
	{
		endpoint: "/api/v1/log",
		method:   GET,
	}, {
		endpoint: "/api/v1/log/entries",
		method:   GET,
		queries:  map[string]string{"logIndex": "10"},
	}, {
		endpoint: "/api/v1/log/proof",
		method:   GET,
		queries:  map[string]string{"firstSize": "10", "lastSize": "20"},
	}, {
		endpoint: "/api/v1/log/entries/retrieve",
		method:   POST,
		body:     "{\"hash\":\"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b\"}",
	}, {
		endpoint: "/api/v1/index/retrieve",
		method:   POST,
		body:     "{\"hash\":\"sha256:2bd37672a9e472c79c64f42b95e362db16870e28a90f3b17fee8faf952e79b4b\"}",
	},
}

var FulcioEndpoints = []ReadProberCheck{
	{
		endpoint: "/api/v1/rootCert",
		method:   GET,
	},
}
