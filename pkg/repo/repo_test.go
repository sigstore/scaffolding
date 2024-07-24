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
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"google.golang.org/protobuf/encoding/protojson"
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

	fulcioRootCertBase64DER = `MIICNzCCAd2gAwIBAgITPLBoBQhl1hqFND9S+SGWbfzaRTAKBggqhkjOPQQDAjBoMQswCQYDVQQGEwJVSzESMBAGA1UECBMJV2lsdHNoaXJlMRMwEQYDVQQHEwpDaGlwcGVuaGFtMQ8wDQYDVQQKEwZSZWRIYXQxDDAKBgNVBAsTA0NUTzERMA8GA1UEAxMIdGVzdGNlcnQwHhcNMjEwMzEyMjMyNDQ5WhcNMzEwMjI4MjMyNDQ5WjBoMQswCQYDVQQGEwJVSzESMBAGA1UECBMJV2lsdHNoaXJlMRMwEQYDVQQHEwpDaGlwcGVuaGFtMQ8wDQYDVQQKEwZSZWRIYXQxDDAKBgNVBAsTA0NUTzERMA8GA1UEAxMIdGVzdGNlcnQwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQRn+Alyof6xP3GQClSwgV0NFuYYEwmKP/WLWr/LwB6LUYzt5v49RlqG83KuaJSpeOj7G7MVABdpIZYWwqAiZV3o2YwZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBATAdBgNVHQ4EFgQUT8Jwm6JuVb0dsiuHUROiHOOVHVkwHwYDVR0jBBgwFoAUT8Jwm6JuVb0dsiuHUROiHOOVHVkwCgYIKoZIzj0EAwIDSAAwRQIhAJkNZmP6sKA+8EebRXFkBa9DPjacBpTcOljJotvKidRhAiAuNrIazKEw2G4dw8x1z6EYk9G+7fJP5m93bjm/JfMBtA==`

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

	ctlogPublicKeyBase64DER = `MIICCgKCAgEAu1Ah4n2P8JGt92Qg86FdR8f1pou43yndggMuRCX0JB+bLn1rUFRAKQVd+xnnd4PXJLLdml8ZohCr0lhBuMxZ7zBzt0T98kblUCxBgABPNpWIkTgacyC8MlIYY/yBSuDWAJOA5IKi4Hh9nI+Mmb/FXgbOz5a5mZx8w7pMiTMu0+Rd9cPzRkUZDQfZsLONr6PwmyCAIL1oK80fevxKZPME0UV8bFPWnRxeVaFr5ddd/DOenV8H6SPyr4ODbSOItpl53y6Az0m3FTIUf8cSsyR7dfE4zpA3M4djjtoKDNFRsTjU2RWVQW9XMaxzznGVGhLEwkC+sYjR5NQvH5iiRvV18q+CGQqNX2+WWM3SPuty3nc86RBNR0FOgSQA0TL2OAs6bJNmfzcwZxAKYbj7/88tj6qrjLaQtFTbBm2a7+TAQfs3UTiQi00zEDYqeSj2WQvacNm1dWEAyx0QNLHiKGTn4TShGj8LUoGyjJ26Y6VPsotvCoj8jM0eaN8Pc9/AYywVI+QktjaPZa7KGH3XJHJkTIQQRcUxOtDstKpcriAefDs8jjL5ju9t5J3qEvgzmclNJKRnla4p3maM0vk+8cC7EXMV4P1zuCwr3akaHFJo5Y0aFhKsnHqTc70LfiFo//8/QsvyjLIUtEWHTkGeuf4PpbYXr5qpJ6tWhG2MARxdeg8CAwEAAQ==`

	rekorPublicKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEF6j2sTItLcs0wKoOpMzI+9lJmCzf
N6mY2prOeaBRV2dnsJzC94hOxkM5pSp9nbAK1TBOI45fOOPsH2rSR++HrA==
-----END PUBLIC KEY-----`

	rekorPublicKeyBase64DER = `MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEF6j2sTItLcs0wKoOpMzI+9lJmCzfN6mY2prOeaBRV2dnsJzC94hOxkM5pSp9nbAK1TBOI45fOOPsH2rSR++HrA==`

	tsaLeafCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`

	tsaLeafCertBase64DER = `MIIB3DCCAWKgAwIBAgIUchkNsH36Xa04b1LqIc+qr9DVecMwCgYIKoZIzj0EAwMwMjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgaW50ZXJtZWRpYXRlMB4XDTIzMDQxNDAwMDAwMFoXDTI0MDQxMzAwMDAwMFowMjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgVGltZXN0YW1waW5nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEUD5ZNbSqYMd6r8qpOOEX9ibGnZT9GsuXOhr/f8U9FJugBGExKYp40OULS0erjZW7xV9xV52NnJf5OeDq4e5ZKqNWMFQwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMIMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAUaW1RudOgVt0leqY0WKYbuPr47wAwCgYIKoZIzj0EAwMDaAAwZQIwbUH9HvD4ejCZJOWQnqAlkqURllvu9M8+VqLbiRK+zSfZCZwsiljRn8MQQRSkXEE5AjEAg+VxqtojfVfu8DhzzhCx9GKETbJHb19iV72mMKUbDAFmzZ6bQ8b54Zb8tidy5aWe`

	tsaIntermedCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`

	tsaIntermedCertBase64DER = `MIICEDCCAZWgAwIBAgIUX8ZO5QXP7vN4dMQ5e9sU3nub8OgwCgYIKoZIzj0EAwMwODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBSb290MB4XDTIzMDQxNDAwMDAwMFoXDTI4MDQxMjAwMDAwMFowMjEVMBMGA1UEChMMR2l0SHViLCBJbmMuMRkwFwYDVQQDExBUU0EgaW50ZXJtZWRpYXRlMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEvMLY/dTVbvIJYANAuszEwJnQE1llftynyMKIMhh48HmqbVr5ygybzsLRLVKbBWOdZ21aeJz+gZiytZetqcyF9WlER5NEMf6JV7ZNojQpxHq4RHGoGSceQv/qvTiZxEDKo2YwZDAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQUaW1RudOgVt0leqY0WKYbuPr47wAwHwYDVR0jBBgwFoAU9NYYlobnAG4c0/qjxyH/lq/wz+QwCgYIKoZIzj0EAwMDaQAwZgIxAK1B185ygCrIYFlIs3GjswjnwSMG6LY8woLVdakKDZxVa8f8cqMs1DhcxJ0+09w95QIxAO+tBzZk7vjUJ9iJgD4R6ZWTxQWKqNm74jO99o+o9sv4FI/SZTZTFyMn0IJEHdNmyA==`

	tsaRootCert = `-----BEGIN CERTIFICATE-----
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

	tsaRootCertBase64DER = `MIIB9DCCAXqgAwIBAgIUa/JAkdUjK4JUwsqtaiRJGWhqLSowCgYIKoZIzj0EAwMwODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBSb290MB4XDTIzMDQxNDAwMDAwMFoXDTMzMDQxMTAwMDAwMFowODEVMBMGA1UEChMMR2l0SHViLCBJbmMuMR8wHQYDVQQDExZJbnRlcm5hbCBTZXJ2aWNlcyBSb290MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEf9jFAXxz4kx68AHRMOkFBhflDcMTvzaXz4x/FCcXjJ/1qEKon/qPIGnaURskDtyNbNDOpeJTDDFqt48iMPrnzpx6IZwqemfUJN4xBEZfza+pYt/iyod+9tZr20RRWSv/o0UwQzAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB/wIBAjAdBgNVHQ4EFgQU9NYYlobnAG4c0/qjxyH/lq/wz+QwCgYIKoZIzj0EAwMDaAAwZQIxALZLZ8BgRXzKxLMMN9VIlO+e4hrBnNBgF7tz7Hnrowv2NetZErIACKFymBlvWDvtMAIwZO+ki6ssQ1bsZo98O8mEAf2NZ7iiCgDDU0Vwjeco6zyeh0zBTs9/7gV6AHNQ53xD`
)

func TestCreateRepo(t *testing.T) {
	files := map[string][]byte{
		"fulcio_v1.crt.pem":          []byte(fulcioRootCert),
		"ctfe.pub":                   []byte(ctlogPublicKey),
		"rekor.pub":                  []byte(rekorPublicKey),
		"tsa_leaf.crt.pem":           []byte(tsaLeafCert),
		"tsa_intermediate_1.crt.pem": []byte(tsaIntermedCert),
		"tsa_root.crt.pem":           []byte(tsaRootCert),
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

func TestCreateRepoWithOptions(t *testing.T) {
	files := map[string][]byte{
		"fulcio_v1.crt.pem":          []byte(fulcioRootCert),
		"ctfe.pub":                   []byte(ctlogPublicKey),
		"rekor.pub":                  []byte(rekorPublicKey),
		"tsa_leaf.crt.pem":           []byte(tsaLeafCert),
		"tsa_intermediate_1.crt.pem": []byte(tsaIntermedCert),
		"tsa_root.crt.pem":           []byte(tsaRootCert),
	}

	repo, dir, err := CreateRepoWithOptions(context.Background(), files, CreateRepoOptions{AddMetadataTargets: true, AddTrustedRoot: true})
	if err != nil {
		t.Fatalf("Failed to CreateRepoWithOptions: %s", err)
	}
	defer os.RemoveAll(dir)
	meta, err := repo.GetMeta()
	if err != nil {
		t.Errorf("Failed to GetMeta: %s", err)
	}
	t.Logf("Got repo meta as: %+v", meta)
	trustedRootContent, err := os.ReadFile(filepath.Join(dir, "repository", "targets", "trusted_root.json"))
	if err != nil {
		t.Errorf("Failed to read trusted_root.json: %s", err)
	}
	t.Logf("Trusted root content: %s", trustedRootContent)

	tr := v1.TrustedRoot{}
	err = protojson.Unmarshal(trustedRootContent, &tr)
	if err != nil {
		t.Errorf("Failed to parse trusted_root.json: %s", err)
	}

	if tr.MediaType != "application/vnd.dev.sigstore.trustedroot+json;version=0.1" {
		t.Errorf("Wrong trusted_root.json mediaType %s", tr.MediaType)
	}
	if base64.StdEncoding.EncodeToString(tr.CertificateAuthorities[0].CertChain.Certificates[0].RawBytes) != fulcioRootCertBase64DER {
		t.Errorf("Wrong fulcio certificate RawBytes")
	}

	if base64.StdEncoding.EncodeToString(tr.TimestampAuthorities[0].CertChain.Certificates[0].RawBytes) != tsaLeafCertBase64DER {
		t.Errorf("Wrong TSA leaf certificate RawBytes")
	}
	if base64.StdEncoding.EncodeToString(tr.TimestampAuthorities[0].CertChain.Certificates[1].RawBytes) != tsaIntermedCertBase64DER {
		t.Errorf("Wrong TSA intermediate certificate RawBytes")
	}
	if base64.StdEncoding.EncodeToString(tr.TimestampAuthorities[0].CertChain.Certificates[2].RawBytes) != tsaRootCertBase64DER {
		t.Errorf("Wrong TSA root certificate RawBytes")
	}

	if base64.StdEncoding.EncodeToString(tr.Tlogs[0].PublicKey.RawBytes) != rekorPublicKeyBase64DER {
		t.Errorf("Wrong tlog (rekor) pubkey RawBytes")
	}

	if base64.StdEncoding.EncodeToString(tr.Ctlogs[0].PublicKey.RawBytes) != ctlogPublicKeyBase64DER {
		t.Errorf("Wrong tlog (rekor) pubkey RawBytes")
	}
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
	rtRekor, err := os.ReadFile(filepath.Join(dstDir, "repository", "targets", "rekor.pub"))
	if err != nil {
		t.Errorf("Failed to read the roundtripped rekor %v", err)
	}
	if !bytes.Equal(files["rekor.pub"], rtRekor) {
		t.Errorf("Roundtripped rekor differs:\n%s\n%s", rekorPublicKey, string(rtRekor))
	}
}
