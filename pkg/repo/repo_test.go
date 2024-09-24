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

package repo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/sigstore/scaffolding/pkg/certs"
	"github.com/stretchr/testify/require"
)

const (
	fulcioRootCert = `-----BEGIN CERTIFICATE-----
MIICNzCCAd2gAwIBAgITPLBoBQhl1hqFND9S+SGWbfzaRTAKBggqhkjOPQQDAjBo
MQswCQYDVQQGEwJVSzESMBAGA1UECBMJV2lsdHNoaXJlMRMwEQYDVQQHEwpDaGlw
cGVuaGFtMQ8wDQYDVQQKEwZSZWRIYXQxDDAKBgNVBAsTA0NUTzERMA8GA1UEAxMI
dGVzdGNlcnQwHhcNMjEwMzEyMjMyNDQ5WhcNMzEwMjI4MjMyNDQ5WjBoMQswCQYD
VQQGEwJVSzESMBAGA1UECBMJV2lsdHNoaXJlMRMwEQYDVQQHEwpDaGlwcGVuaGFt
MQ8wDQYDVQQKEwZSZWRIYXQxDDAKBgNVBAsTA0NUTzERMA8GA1UEAxMIdGVzdGNl
cnQwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQRn+Alyof6xP3GQClSwgV0NFuY
YEwmKP/WLWr/LwB6LUYzt5v49RlqG83KuaJSpeOj7G7MVABdpIZYWwqAiZV3o2Yw
ZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBATAdBgNVHQ4EFgQU
T8Jwm6JuVb0dsiuHUROiHOOVHVkwHwYDVR0jBBgwFoAUT8Jwm6JuVb0dsiuHUROi
HOOVHVkwCgYIKoZIzj0EAwIDSAAwRQIhAJkNZmP6sKA+8EebRXFkBa9DPjacBpTc
OljJotvKidRhAiAuNrIazKEw2G4dw8x1z6EYk9G+7fJP5m93bjm/JfMBtA==
-----END CERTIFICATE-----`

	ctlogPublicKey = `-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEAu1Ah4n2P8JGt92Qg86FdR8f1pou43yndggMuRCX0JB+bLn1rUFRA
KQVd+xnnd4PXJLLdml8ZohCr0lhBuMxZ7zBzt0T98kblUCxBgABPNpWIkTgacyC8
MlIYY/yBSuDWAJOA5IKi4Hh9nI+Mmb/FXgbOz5a5mZx8w7pMiTMu0+Rd9cPzRkUZ
DQfZsLONr6PwmyCAIL1oK80fevxKZPME0UV8bFPWnRxeVaFr5ddd/DOenV8H6SPy
r4ODbSOItpl53y6Az0m3FTIUf8cSsyR7dfE4zpA3M4djjtoKDNFRsTjU2RWVQW9X
MaxzznGVGhLEwkC+sYjR5NQvH5iiRvV18q+CGQqNX2+WWM3SPuty3nc86RBNR0FO
gSQA0TL2OAs6bJNmfzcwZxAKYbj7/88tj6qrjLaQtFTbBm2a7+TAQfs3UTiQi00z
EDYqeSj2WQvacNm1dWEAyx0QNLHiKGTn4TShGj8LUoGyjJ26Y6VPsotvCoj8jM0e
aN8Pc9/AYywVI+QktjaPZa7KGH3XJHJkTIQQRcUxOtDstKpcriAefDs8jjL5ju9t
5J3qEvgzmclNJKRnla4p3maM0vk+8cC7EXMV4P1zuCwr3akaHFJo5Y0aFhKsnHqT
c70LfiFo//8/QsvyjLIUtEWHTkGeuf4PpbYXr5qpJ6tWhG2MARxdeg8CAwEAAQ==
-----END RSA PUBLIC KEY-----`

	rekorPublicKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEF6j2sTItLcs0wKoOpMzI+9lJmCzf
N6mY2prOeaBRV2dnsJzC94hOxkM5pSp9nbAK1TBOI45fOOPsH2rSR++HrA==
-----END PUBLIC KEY-----`

	tsaCertChain = `-----BEGIN CERTIFICATE-----
MIIB3DCCAWKgAwIBAgIUchkNsH36Xa04b1LqIc+qr9DVecMwCgYIKoZIzj0EAwMw
MjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgaW50ZXJtZWRp
YXRlMB4XDTIzMDQxNDAwMDAwMFoXDTI0MDQxMzAwMDAwMFowMjEVMBMGA1UEChMM
R2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgVGltZXN0YW1waW5nMFkwEwYHKoZI
zj0CAQYIKoZIzj0DAQcDQgAEUD5ZNbSqYMd6r8qpOOEX9ibGnZT9GsuXOhr/f8U9
FJugBGExKYp40OULS0erjZW7xV9xV52NnJf5OeDq4e5ZKqNWMFQwDgYDVR0PAQH/
BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMIMAwGA1UdEwEB/wQCMAAwHwYDVR0j
BBgwFoAUaW1RudOgVt0leqY0WKYbuPr47wAwCgYIKoZIzj0EAwMDaAAwZQIwbUH9
HvD4ejCZJOWQnqAlkqURllvu9M8+VqLbiRK+zSfZCZwsiljRn8MQQRSkXEE5AjEA
g+VxqtojfVfu8DhzzhCx9GKETbJHb19iV72mMKUbDAFmzZ6bQ8b54Zb8tidy5aWe
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIICEDCCAZWgAwIBAgIUX8ZO5QXP7vN4dMQ5e9sU3nub8OgwCgYIKoZIzj0EAwMw
ODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2
aWNlcyBSb290MB4XDTIzMDQxNDAwMDAwMFoXDTI4MDQxMjAwMDAwMFowMjEVMBMG
A1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgaW50ZXJtZWRpYXRlMHYw
EAYHKoZIzj0CAQYFK4EEACIDYgAEvMLY/dTVbvIJYANAuszEwJnQE1llftynyMKI
Mhh48HmqbVr5ygybzsLRLVKbBWOdZ21aeJz+gZiytZetqcyF9WlER5NEMf6JV7ZN
ojQpxHq4RHGoGSceQv/qvTiZxEDKo2YwZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0T
AQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQUaW1RudOgVt0leqY0WKYbuPr47wAwHwYD
VR0jBBgwFoAU9NYYlobnAG4c0/qjxyH/lq/wz+QwCgYIKoZIzj0EAwMDaQAwZgIx
AK1B185ygCrIYFlIs3GjswjnwSMG6LY8woLVdakKDZxVa8f8cqMs1DhcxJ0+09w9
5QIxAO+tBzZk7vjUJ9iJgD4R6ZWTxQWKqNm74jO99o+o9sv4FI/SZTZTFyMn0IJE
HdNmyA==
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIB9DCCAXqgAwIBAgIUa/JAkdUjK4JUwsqtaiRJGWhqLSowCgYIKoZIzj0EAwMw
ODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2
aWNlcyBSb290MB4XDTIzMDQxNDAwMDAwMFoXDTMzMDQxMTAwMDAwMFowODEVMBMG
A1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBS
b290MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEf9jFAXxz4kx68AHRMOkFBhflDcMT
vzaXz4x/FCcXjJ/1qEKon/qPIGnaURskDtyNbNDOpeJTDDFqt48iMPrnzpx6IZwq
emfUJN4xBEZfza+pYt/iyod+9tZr20RRWSv/o0UwQzAOBgNVHQ8BAf8EBAMCAQYw
EgYDVR0TAQH/BAgwBgEB/wIBAjAdBgNVHQ4EFgQU9NYYlobnAG4c0/qjxyH/lq/w
z+QwCgYIKoZIzj0EAwMDaAAwZQIxALZLZ8BgRXzKxLMMN9VIlO+e4hrBnNBgF7tz
7Hnrowv2NetZErIACKFymBlvWDvtMAIwZO+ki6ssQ1bsZo98O8mEAf2NZ7iiCgDD
U0Vwjeco6zyeh0zBTs9/7gV6AHNQ53xD
-----END CERTIFICATE-----`

	trJSONValidForStart = "2024-08-26T12:03:30.241272895Z"
	trJSON              = `{
  "mediaType": "application/vnd.dev.sigstore.trustedroot+json;version=0.1",
  "tlogs": [
    {
      "hashAlgorithm": "SHA2_256",
      "publicKey": {
        "rawBytes": "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEF6j2sTItLcs0wKoOpMzI+9lJmCzfN6mY2prOeaBRV2dnsJzC94hOxkM5pSp9nbAK1TBOI45fOOPsH2rSR++HrA==",
        "keyDetails": "PKIX_ECDSA_P256_SHA_256",
        "validFor": {
          "start": "2024-08-26T12:03:30.241272895Z"
        }
      },
      "logId": {
        "keyId": "xBzny6gmou42sCYrHOzNuGqi1s2cMxcCEq1wrKF9XDs="
      }
    }
  ],
  "certificateAuthorities": [
    {
      "subject": {
        "organization": "RedHat",
        "commonName": "testcert"
      },
      "certChain": {
        "certificates": [
          {
            "rawBytes": "MIICNzCCAd2gAwIBAgITPLBoBQhl1hqFND9S+SGWbfzaRTAKBggqhkjOPQQDAjBoMQswCQYDVQQGEwJVSzESMBAGA1UECBMJV2lsdHNoaXJlMRMwEQYDVQQHEwpDaGlwcGVuaGFtMQ8wDQYDVQQKEwZSZWRIYXQxDDAKBgNVBAsTA0NUTzERMA8GA1UEAxMIdGVzdGNlcnQwHhcNMjEwMzEyMjMyNDQ5WhcNMzEwMjI4MjMyNDQ5WjBoMQswCQYDVQQGEwJVSzESMBAGA1UECBMJV2lsdHNoaXJlMRMwEQYDVQQHEwpDaGlwcGVuaGFtMQ8wDQYDVQQKEwZSZWRIYXQxDDAKBgNVBAsTA0NUTzERMA8GA1UEAxMIdGVzdGNlcnQwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQRn+Alyof6xP3GQClSwgV0NFuYYEwmKP/WLWr/LwB6LUYzt5v49RlqG83KuaJSpeOj7G7MVABdpIZYWwqAiZV3o2YwZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBATAdBgNVHQ4EFgQUT8Jwm6JuVb0dsiuHUROiHOOVHVkwHwYDVR0jBBgwFoAUT8Jwm6JuVb0dsiuHUROiHOOVHVkwCgYIKoZIzj0EAwIDSAAwRQIhAJkNZmP6sKA+8EebRXFkBa9DPjacBpTcOljJotvKidRhAiAuNrIazKEw2G4dw8x1z6EYk9G+7fJP5m93bjm/JfMBtA=="
          }
        ]
      },
      "validFor": {
        "start": "2021-03-12T23:24:49Z",
        "end": "2031-02-28T23:24:49Z"
      }
    }
  ],
  "ctlogs": [
    {
      "hashAlgorithm": "SHA2_256",
      "publicKey": {
        "rawBytes": "MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAu1Ah4n2P8JGt92Qg86FdR8f1pou43yndggMuRCX0JB+bLn1rUFRAKQVd+xnnd4PXJLLdml8ZohCr0lhBuMxZ7zBzt0T98kblUCxBgABPNpWIkTgacyC8MlIYY/yBSuDWAJOA5IKi4Hh9nI+Mmb/FXgbOz5a5mZx8w7pMiTMu0+Rd9cPzRkUZDQfZsLONr6PwmyCAIL1oK80fevxKZPME0UV8bFPWnRxeVaFr5ddd/DOenV8H6SPyr4ODbSOItpl53y6Az0m3FTIUf8cSsyR7dfE4zpA3M4djjtoKDNFRsTjU2RWVQW9XMaxzznGVGhLEwkC+sYjR5NQvH5iiRvV18q+CGQqNX2+WWM3SPuty3nc86RBNR0FOgSQA0TL2OAs6bJNmfzcwZxAKYbj7/88tj6qrjLaQtFTbBm2a7+TAQfs3UTiQi00zEDYqeSj2WQvacNm1dWEAyx0QNLHiKGTn4TShGj8LUoGyjJ26Y6VPsotvCoj8jM0eaN8Pc9/AYywVI+QktjaPZa7KGH3XJHJkTIQQRcUxOtDstKpcriAefDs8jjL5ju9t5J3qEvgzmclNJKRnla4p3maM0vk+8cC7EXMV4P1zuCwr3akaHFJo5Y0aFhKsnHqTc70LfiFo//8/QsvyjLIUtEWHTkGeuf4PpbYXr5qpJ6tWhG2MARxdeg8CAwEAAQ==",
        "keyDetails": "PKIX_RSA_PKCS1V15_4096_SHA256",
        "validFor": {
          "start": "2024-08-26T12:03:30.241272895Z"
        }
      },
      "logId": {
        "keyId": "G3CTL21UG8/5ygV+/WVy/pvB8nUiZGOEnMVKIEDzPxY="
      }
    }
  ],
  "timestampAuthorities": [
    {
      "subject": {
        "organization": "GitHub, Inc.",
        "commonName": "Internal Services Root"
      },
      "certChain": {
        "certificates": [
          {
            "rawBytes": "MIIB3DCCAWKgAwIBAgIUchkNsH36Xa04b1LqIc+qr9DVecMwCgYIKoZIzj0EAwMwMjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgaW50ZXJtZWRpYXRlMB4XDTIzMDQxNDAwMDAwMFoXDTI0MDQxMzAwMDAwMFowMjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgVGltZXN0YW1waW5nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEUD5ZNbSqYMd6r8qpOOEX9ibGnZT9GsuXOhr/f8U9FJugBGExKYp40OULS0erjZW7xV9xV52NnJf5OeDq4e5ZKqNWMFQwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMIMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUaW1RudOgVt0leqY0WKYbuPr47wAwCgYIKoZIzj0EAwMDaAAwZQIwbUH9HvD4ejCZJOWQnqAlkqURllvu9M8+VqLbiRK+zSfZCZwsiljRn8MQQRSkXEE5AjEAg+VxqtojfVfu8DhzzhCx9GKETbJHb19iV72mMKUbDAFmzZ6bQ8b54Zb8tidy5aWe"
          },
          {
            "rawBytes": "MIICEDCCAZWgAwIBAgIUX8ZO5QXP7vN4dMQ5e9sU3nub8OgwCgYIKoZIzj0EAwMwODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBSb290MB4XDTIzMDQxNDAwMDAwMFoXDTI4MDQxMjAwMDAwMFowMjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgaW50ZXJtZWRpYXRlMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEvMLY/dTVbvIJYANAuszEwJnQE1llftynyMKIMhh48HmqbVr5ygybzsLRLVKbBWOdZ21aeJz+gZiytZetqcyF9WlER5NEMf6JV7ZNojQpxHq4RHGoGSceQv/qvTiZxEDKo2YwZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQUaW1RudOgVt0leqY0WKYbuPr47wAwHwYDVR0jBBgwFoAU9NYYlobnAG4c0/qjxyH/lq/wz+QwCgYIKoZIzj0EAwMDaQAwZgIxAK1B185ygCrIYFlIs3GjswjnwSMG6LY8woLVdakKDZxVa8f8cqMs1DhcxJ0+09w95QIxAO+tBzZk7vjUJ9iJgD4R6ZWTxQWKqNm74jO99o+o9sv4FI/SZTZTFyMn0IJEHdNmyA=="
          },
          {
            "rawBytes": "MIIB9DCCAXqgAwIBAgIUa/JAkdUjK4JUwsqtaiRJGWhqLSowCgYIKoZIzj0EAwMwODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBSb290MB4XDTIzMDQxNDAwMDAwMFoXDTMzMDQxMTAwMDAwMFowODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBSb290MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEf9jFAXxz4kx68AHRMOkFBhflDcMTvzaXz4x/FCcXjJ/1qEKon/qPIGnaURskDtyNbNDOpeJTDDFqt48iMPrnzpx6IZwqemfUJN4xBEZfza+pYt/iyod+9tZr20RRWSv/o0UwQzAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBAjAdBgNVHQ4EFgQU9NYYlobnAG4c0/qjxyH/lq/wz+QwCgYIKoZIzj0EAwMDaAAwZQIxALZLZ8BgRXzKxLMMN9VIlO+e4hrBnNBgF7tz7Hnrowv2NetZErIACKFymBlvWDvtMAIwZO+ki6ssQ1bsZo98O8mEAf2NZ7iiCgDDU0Vwjeco6zyeh0zBTs9/7gV6AHNQ53xD"
          }
        ]
      },
      "validFor": {
        "start": "2023-04-14T00:00:00Z",
        "end": "2024-04-13T00:00:00Z"
      }
    }
  ]
}`
)

func TestCreateRepo(t *testing.T) {
	files := map[string][]byte{
		"fulcio_v1.crt.pem": []byte(fulcioRootCert),
		"ctfe.pub":          []byte(ctlogPublicKey),
		"rekor.pub":         []byte(rekorPublicKey),
	}
	repo, dir, err := CreateRepo(context.Background(), files)
	if err != nil {
		t.Fatalf("Failed to CreateRepo: %s", err)
	}
	defer os.RemoveAll(dir)
	meta, err := repo.GetMeta()
	if err != nil {
		t.Errorf("Failed to GetMeta: %s", err)
	}
	t.Logf("Got repo meta as: %+v", meta)
}

func TestCompressUncompressFS(t *testing.T) {
	files := map[string][]byte{
		"fulcio_v1.crt.pem": []byte(fulcioRootCert),
		"ctfe.pub":          []byte(ctlogPublicKey),
		"rekor.pub":         []byte(rekorPublicKey),
	}
	repo, dir, err := CreateRepo(context.Background(), files)
	if err != nil {
		t.Fatalf("Failed to CreateRepo: %s", err)
	}
	defer os.RemoveAll(dir)

	var buf bytes.Buffer
	fsys := os.DirFS(dir)
	if err = CompressFS(fsys, &buf, map[string]bool{"keys": true, "staged": true}); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}
	// #nosec G306 -- test
	if err := os.WriteFile(filepath.Join(t.TempDir(), "newcompressed"), buf.Bytes(), os.ModePerm); err != nil {
		t.Fatalf("Failed to write compressed output")
	}
	dstDir := t.TempDir()
	if err = Uncompress(&buf, dstDir); err != nil {
		t.Fatalf("Failed to uncompress: %v", err)
	}
	// Then check that files have been uncompressed there.
	meta, err := repo.GetMeta()
	if err != nil {
		t.Errorf("Failed to GetMeta: %s", err)
	}
	root := meta["root.json"]

	// This should have roundtripped to the new directory.
	rtRoot, err := os.ReadFile(filepath.Join(dstDir, "repository", "root.json"))
	if err != nil {
		t.Errorf("Failed to read the roundtripped root %v", err)
	}
	if !bytes.Equal(root, rtRoot) {
		t.Errorf("Roundtripped root differs:\n%s\n%s", string(root), string(rtRoot))
	}

	// As well as, say rekor.pub under targets dir
	rtRekor, err := os.ReadFile(
		filepath.Join(
			dstDir,
			"repository",
			"targets",
			"6a0d8ce486b4aa7a2b30bf0940a879a00d1bb2738c4874543e179781d42b8ea1d12f34698dc95b3beeb9785036f7f2272e2fc92b994a04ac59ae458b5f4a2a7c.rekor.pub",
		),
	)
	if err != nil {
		t.Errorf("Failed to read the roundtripped rekor %v", err)
	}
	if !bytes.Equal(files["rekor.pub"], rtRekor) {
		t.Errorf("Roundtripped rekor differs:\n%s\n%s", rekorPublicKey, string(rtRekor))
	}
}

func TestConstructTrustedRoot(t *testing.T) {
	tsaCerts, err := certs.SplitCertChain([]byte(tsaCertChain), "tsa")
	if err != nil {
		t.Fatalf("Failed to split TSA cert chain: %v", err)
	}

	targets := []TargetWithMetadata{
		{Name: "fulcio.crt.pem", Bytes: []byte(fulcioRootCert)},
		{Name: "ctfe.pub", Bytes: []byte(ctlogPublicKey)},
		{Name: "rekor.pub", Bytes: []byte(rekorPublicKey)},
	}

	for k, v := range tsaCerts {
		targets = append(targets, TargetWithMetadata{Name: k, Bytes: v})
	}

	tr, err := constructTrustedRoot(targets)
	if err != nil {
		t.Fatalf("Failed to construct trusted root: %v", err)
	}
	if tr.Name != "trusted_root.json" {
		t.Fatalf("Wrong name for trusted root target. Expected 'trusted_root.json', got '%s'", tr.Name)
	}
	actualTr := string(tr.Bytes)

	// the "validFor" for tlog/ctlog keys will be current time; let's replace them
	// to be the time from the expected sample
	re := regexp.MustCompile(`"validFor"\s*:\s*\{\s*"start"\s*:\s*"[^"]+"\s*\}`)
	actualTr = re.ReplaceAllString(actualTr, fmt.Sprintf(`"validFor": {"start": "%s"}`, trJSONValidForStart))

	require.JSONEq(t, trJSON, actualTr)
}
