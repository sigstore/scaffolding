// Copyright 2022 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package certs

import (
	"bytes"
	"strings"
	"testing"
)

const (
	chain = `
-----BEGIN CERTIFICATE-----
MIIBzDCCAXKgAwIBAgIUfyGKDoFa7y6s/W1p1CiTmBRs1eAwCgYIKoZIzj0EAwIw
MDEOMAwGA1UEChMFbG9jYWwxHjAcBgNVBAMTFVRlc3QgVFNBIEludGVybWVkaWF0
ZTAeFw0yMjExMDkyMDMxMzRaFw0zMTExMDkyMDM0MzRaMDAxDjAMBgNVBAoTBWxv
Y2FsMR4wHAYDVQQDExVUZXN0IFRTQSBUaW1lc3RhbXBpbmcwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAAR3KcDy9jwARX0rDvyr+MGGkG3n1OA0MU5+ZiDmgusFyk6U
6bovKWVMfD8J8NTcJZE0RaYJr8/dE9kgcIIXlhMwo2owaDAOBgNVHQ8BAf8EBAMC
B4AwHQYDVR0OBBYEFHNn5R3b3MtUdSNrFO49Q6XDVSnkMB8GA1UdIwQYMBaAFNLS
6gno7Om++Qt5zIa+H9o0HiT2MBYGA1UdJQEB/wQMMAoGCCsGAQUFBwMIMAoGCCqG
SM49BAMCA0gAMEUCIQCF0olohnvdUq6T7/wPk19Z5aQP/yxRTjCWYuhn/TCyHgIg
azV3air4GRZbN9bdYtcQ7JUAKq89GOhtFfl6kcoVUvU=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIB0jCCAXigAwIBAgIUXpBmYJFFaGW3cC8p6b/DHr1i8IowCgYIKoZIzj0EAwIw
KDEOMAwGA1UEChMFbG9jYWwxFjAUBgNVBAMTDVRlc3QgVFNBIFJvb3QwHhcNMjIx
MTA5MjAyOTM0WhcNMzIxMTA5MjAzNDM0WjAwMQ4wDAYDVQQKEwVsb2NhbDEeMBwG
A1UEAxMVVGVzdCBUU0EgSW50ZXJtZWRpYXRlMFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAEKDPDRIwDS1ZCymub6yanCG5ma0qDjLpNonDvooSkRHEgU0TNibeJn6M+
5W608hCw8nwuucMbXQ41kNeuBeevyqN4MHYwDgYDVR0PAQH/BAQDAgEGMBMGA1Ud
JQQMMAoGCCsGAQUFBwMIMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFNLS6gno
7Om++Qt5zIa+H9o0HiT2MB8GA1UdIwQYMBaAFB1nvXpNK7AuQlbJ+ya6nPSqWi+T
MAoGCCqGSM49BAMCA0gAMEUCIGiwqCI29w7C4V8TltCsi728s5DtklCPySDASUSu
a5y5AiEA40Ifdlwf7Uj8q8NSD6Z4g/0js0tGNdLSUJ1do/WoN0s=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBlDCCATqgAwIBAgIUYZx9sS14En7SuHDOJJP4IPopMjUwCgYIKoZIzj0EAwIw
KDEOMAwGA1UEChMFbG9jYWwxFjAUBgNVBAMTDVRlc3QgVFNBIFJvb3QwHhcNMjIx
MTA5MjAyOTM0WhcNMzIxMTA5MjAzNDM0WjAoMQ4wDAYDVQQKEwVsb2NhbDEWMBQG
A1UEAxMNVGVzdCBUU0EgUm9vdDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABAbB
B0SU8G75hVIUphChA4nfOwNWP347TjScIdsEPrKVn+/Y1HmmLHJDjSfn+xhEFoEk
7jqgrqon48i4xbo7xAujQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTAD
AQH/MB0GA1UdDgQWBBQdZ716TSuwLkJWyfsmupz0qlovkzAKBggqhkjOPQQDAgNI
ADBFAiBe5P56foqmFcZAVpEeAOFZrAlEiq05CCpMNYh5EjLvmAIhAKNF6xIV5uFd
pSTJsAwzjW78CKQm7qol0uPmPPu6mNaw
-----END CERTIFICATE-----
`
	root = `-----BEGIN CERTIFICATE-----
MIIBlDCCATqgAwIBAgIUYZx9sS14En7SuHDOJJP4IPopMjUwCgYIKoZIzj0EAwIw
KDEOMAwGA1UEChMFbG9jYWwxFjAUBgNVBAMTDVRlc3QgVFNBIFJvb3QwHhcNMjIx
MTA5MjAyOTM0WhcNMzIxMTA5MjAzNDM0WjAoMQ4wDAYDVQQKEwVsb2NhbDEWMBQG
A1UEAxMNVGVzdCBUU0EgUm9vdDBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABAbB
B0SU8G75hVIUphChA4nfOwNWP347TjScIdsEPrKVn+/Y1HmmLHJDjSfn+xhEFoEk
7jqgrqon48i4xbo7xAujQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTAD
AQH/MB0GA1UdDgQWBBQdZ716TSuwLkJWyfsmupz0qlovkzAKBggqhkjOPQQDAgNI
ADBFAiBe5P56foqmFcZAVpEeAOFZrAlEiq05CCpMNYh5EjLvmAIhAKNF6xIV5uFd
pSTJsAwzjW78CKQm7qol0uPmPPu6mNaw
-----END CERTIFICATE-----
`
	intermediate = `-----BEGIN CERTIFICATE-----
MIIB0jCCAXigAwIBAgIUXpBmYJFFaGW3cC8p6b/DHr1i8IowCgYIKoZIzj0EAwIw
KDEOMAwGA1UEChMFbG9jYWwxFjAUBgNVBAMTDVRlc3QgVFNBIFJvb3QwHhcNMjIx
MTA5MjAyOTM0WhcNMzIxMTA5MjAzNDM0WjAwMQ4wDAYDVQQKEwVsb2NhbDEeMBwG
A1UEAxMVVGVzdCBUU0EgSW50ZXJtZWRpYXRlMFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAEKDPDRIwDS1ZCymub6yanCG5ma0qDjLpNonDvooSkRHEgU0TNibeJn6M+
5W608hCw8nwuucMbXQ41kNeuBeevyqN4MHYwDgYDVR0PAQH/BAQDAgEGMBMGA1Ud
JQQMMAoGCCsGAQUFBwMIMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFNLS6gno
7Om++Qt5zIa+H9o0HiT2MB8GA1UdIwQYMBaAFB1nvXpNK7AuQlbJ+ya6nPSqWi+T
MAoGCCqGSM49BAMCA0gAMEUCIGiwqCI29w7C4V8TltCsi728s5DtklCPySDASUSu
a5y5AiEA40Ifdlwf7Uj8q8NSD6Z4g/0js0tGNdLSUJ1do/WoN0s=
-----END CERTIFICATE-----
`
	leaf = `-----BEGIN CERTIFICATE-----
MIIBzDCCAXKgAwIBAgIUfyGKDoFa7y6s/W1p1CiTmBRs1eAwCgYIKoZIzj0EAwIw
MDEOMAwGA1UEChMFbG9jYWwxHjAcBgNVBAMTFVRlc3QgVFNBIEludGVybWVkaWF0
ZTAeFw0yMjExMDkyMDMxMzRaFw0zMTExMDkyMDM0MzRaMDAxDjAMBgNVBAoTBWxv
Y2FsMR4wHAYDVQQDExVUZXN0IFRTQSBUaW1lc3RhbXBpbmcwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAAR3KcDy9jwARX0rDvyr+MGGkG3n1OA0MU5+ZiDmgusFyk6U
6bovKWVMfD8J8NTcJZE0RaYJr8/dE9kgcIIXlhMwo2owaDAOBgNVHQ8BAf8EBAMC
B4AwHQYDVR0OBBYEFHNn5R3b3MtUdSNrFO49Q6XDVSnkMB8GA1UdIwQYMBaAFNLS
6gno7Om++Qt5zIa+H9o0HiT2MBYGA1UdJQEB/wQMMAoGCCsGAQUFBwMIMAoGCCqG
SM49BAMCA0gAMEUCIQCF0olohnvdUq6T7/wPk19Z5aQP/yxRTjCWYuhn/TCyHgIg
azV3air4GRZbN9bdYtcQ7JUAKq89GOhtFfl6kcoVUvU=
-----END CERTIFICATE-----
`
)

var want = map[string][]byte{
	"tsa_leaf.crt.pem":           []byte(leaf),
	"tsa_intermediate_0.crt.pem": []byte(intermediate),
	"tsa_root.crt.pem":           []byte(root),
}

func TestSplitCertChain(t *testing.T) {
	got, err := SplitCertChain([]byte(chain), "tsa")
	if err != nil {
		t.Fatalf("SplitCertChain failed: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("Did not get right number of pieces want 3 got %d", len(got))
	}
	for k, v := range want {
		if !bytes.Equal(got[k], v) {
			t.Errorf("Certs differ for %s want\n%q\n got\n%q\n", k, string(v), string(got[k]))
		}
	}
}

func TestSplitCertChainErrors(t *testing.T) {
	wantErr := "error during PEM decoding"
	_, err := SplitCertChain([]byte("hello world"), "tsa")
	if err == nil {
		t.Fatalf("did not get error when wanted one")
	}
	if !strings.Contains(err.Error(), wantErr) {
		t.Errorf("Unexpected error message, want: %s got: %s", wantErr, err.Error())
	}
}
