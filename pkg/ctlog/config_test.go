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

package ctlog

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/pem"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sigstore/rekor/pkg/pki/x509/testutils"
)

// Just a test Root Cert from a Fulcio instance spun up using Scaffolding.
const (
	existingRootCert = `-----BEGIN CERTIFICATE-----
MIIFwzCCA6ugAwIBAgIIROLjjjoc1aowDQYJKoZIhvcNAQELBQAwfjEMMAoGA1UE
BhMDVVNBMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNp
c2NvMRYwFAYDVQQJEw01NDggTWFya2V0IFN0MQ4wDAYDVQQREwU1NzI3NDEZMBcG
A1UEChMQTGludXggRm91bmRhdGlvbjAeFw0yMjA4MTkxMDIwMDNaFw0yMzA4MTkx
MDIwMDNaMH4xDDAKBgNVBAYTA1VTQTETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQG
A1UEBxMNU2FuIEZyYW5jaXNjbzEWMBQGA1UECRMNNTQ4IE1hcmtldCBTdDEOMAwG
A1UEERMFNTcyNzQxGTAXBgNVBAoTEExpbnV4IEZvdW5kYXRpb24wggIiMA0GCSqG
SIb3DQEBAQUAA4ICDwAwggIKAoICAQDx2lkTbOHD6Rm1tGaU1oBOxfjiehkAtvkS
rjgg8Ba+HxbsCHpkUCWra659IgFKq+TO2EIT8YlXQ3srqTuSTW6xAcezUvCJCb/g
m+muUBomBTXCAUn1TBmcv3dV77a1c6ODkUeUnKLYamEJyOWrsJLvOY1+xLp7ugR8
wOnfGipIheCytJb728Yq7X8hAN9VfYoeYRY7iVEqQUPdkg3TZYbxqeVa0j9dmWvr
2WxgHFHgmPmqfttX0AHRRIfcOH60ZgHx8PllEQAckpGT0dStGtF5s66W/uPyN0KK
TulUijQ4h5vuBuxP3QecQBqpSfs0TIzkYwNLOycTzXh32j4bdvSNs7/7XCsEpF5l
kdjzNcpWLu2nMyRR33mIDo9Dxxa/dJNBDfX3s0GRn4qD5IW8IKKbqJVRyEG/xHiF
xtKXkiWP0PlEptwfIpx75NvcWlfwQHYLk5+1f/fv3RBkirHpKUAFL+zVf55H/WVM
X5WmZsjSqcAbfJYYj6L8+i4J6NsFvnuMu7Dvaq0RCgImvYEPMr6XOzC6luOgkoeq
cGhkoANrLq7qeGHjFsSbWJ5jUvCIlbIL/kjWbMP3f3yR8aqeWdKltK5FPcVbtFRB
rQSExjWu3sKth7koSyvDSWKkf+ZygWKWCd8Pu/MTOX1yW7OzDUSmBwRFzvTbm0Y0
x7hDqH2/owIDAQABo0UwQzAOBgNVHQ8BAf8EBAMCAQYwEgYDVR0TAQH/BAgwBgEB
/wIBATAdBgNVHQ4EFgQUv5NQppUCD9b1w7xLHUdnKjqoBBYwDQYJKoZIhvcNAQEL
BQADggIBADGZ7o0b8MSIoMLRTrV29fChVYVmZ/yFR7Fp8iLXzh4d0FCc83b6dm1E
hsTRxt3OMxiNYeKm5CgrAgQdHPC2+s4XupPexPHnHA4/vfjEAoZfW8zQYmtr7DsW
mXy5W6o7yR7OOfNsGJfK+jHiWZQ/FSQuzkvPhhhljUNWrgdusfediYKaO0r7Ipa4
1uNd8BzdyucRNTDzXfIVNcEWX0Xbx+O2CsJbfN0V/npJSHdaT7v2pVUmhJOu8o9G
Yy8IaXznHyRJy0DRVbTjhHV2+Fx9zFAG61ftUxMdFEvugbLzMVDVWX1JjbbBuhRL
qZB+TitNFEbcZIrFAA1VRUkSRUU9d6/PPgbvhwhANKjdA5EVXMeinuDqdlNNGbbz
uWCwOn8kl/MUxppnAfKE1h76UtOckVszal3MiejvgXx6Zo8CuYfTaHLbEvR8+Dk7
kSeYUuRUF93CTO0MIgz/t+igrrhmwbXSUAgMxLWZB/WMHBmX/N3TsKp+UiwHM9MH
GGAJmZL9EFfEmELHi1+ygSM2QxjRSzcPk1oEZeHY/PyTyFIu1X/HSZW8i9m5VOfy
4Mac/kz73BN6BwM/me2yoyF2jm+mhBgM57Z8z4mZDXgrBIsK9d4o7GMJTronAv8a
KTkomoSY/OxE/5doBCACehThH+96joWfgC0rXi9qAwZ6hwIMJAKy
-----END CERTIFICATE-----
`

	// ECDSA private key
	privateKeyEncodedECDSA = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSURSVVYwbHhqMGE5eXdXUUdURUlCT2FDdVo5amNCYmJFS3puL09zaldFKzBvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFdUp5eU0wL3BPUm1rRVVUTzdwMlNnQ0VrV3M2WWo2VHNRb0Y3eDM3QWtpSXEvQ3llaFNveQpOSjFaZy9YQkduaXpNNHZhSk12MXZDdGFDR0x2RGdGd1lRPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="

	// ECDSA public key
	publicKeyEncodedECDSA = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFdUp5eU0wL3BPUm1rRVVUTzdwMlNnQ0VrV3M2WQpqNlRzUW9GN3gzN0FraUlxL0N5ZWhTb3lOSjFaZy9YQkduaXpNNHZhSk12MXZDdGFDR0x2RGdGd1lRPT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="

	privateKeyEncodedRSA = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUpRZ0lCQURBTkJna3Foa2lHOXcwQkFRRUZBQVNDQ1N3d2dna29BZ0VBQW9JQ0FRRFNQM1c2ekFUdDZuWTUKbVM3dU5Qeks3MHRHUWJaWlRzWllkSEUxUGpLcGdtT1JuZFdtc1pNN2RuYkNGWFllN29weFFpL2hyclJ3UngwbwoyWjhOM3E2L1o3NGliT1ZDa2JkU2xpUW9ROVhKYUlRMy8zOHJKandTWVQ1NG5ReU8ydStMYUwyTWc4RFIrUmhHCjRoclJCSzB3ZEc4N0I3RkVrbUordDNJY0tURmJweHRTOE0yam5NNGM4eHhiWDMvUGNKY2tPc1ZXdlNIRCs5N2UKVWJZYURsbmc1anJJeTdHK29CVGVTMnkyWE9NYURyd1lHNHV3T3pLV0hzYTZxUGlMWEdoOFF4Y25UMFo3QWdMcAptOTBtQWp0L2hyeGVqb3JrdXp2N2tXTWhpZ0VGSjlXd0sreXJzM3JYNFYyTnlCUXRpeTBmU0gzVjFWV1ZkSDZpCmZvcHVKRGppUXB0WmF1Mk4vT3B5cURWdlZmVU8rWlVjWmNZSXJtaVF3N1JFNjUyZ0pwMUlucTQrRU9Ob2tvc2IKRTIwVFEycmhZRUlWK2pYMUMvOXMvMisxUlF0dktPeHVCZCtpWXZZaFRONm4zTFZKRi9nZUxGbjVhUTNEdmk5OQo0ZHkrU2RXSnU2ekU5aHlYS0RMRCtoZENtUk4rSWpTcGw4bFJicFdjQVRoeDhtbmxpSWRqWFU5cDkvd2QyY1J3Ck1YN211VHFDSU0wc1Q2SXQ0R3VZdWNVOGIycHNPcUp4azU2SExueUtvdlZoQmlTT3lIemk5b1RPQnFCdlRyV2MKMCtMS1lub2diUFRpR2MxMmQvcU85VGFvcngzZDdSMkIzWmpzTTRaRlJld2tSUko4Vmw2WStrZ3ZYcEdyczR1Wgp1TkF0Ykd6a1B4UG5xZEJqT2U4aU5RaVdrMGprZVFJREFRQUJBb0lDQUFPZmZGUGZ1Q1lHdmFBaklrcVFqSVZOCllsVmFSSWpTeHJBM2hzdml2OVMrbm5YMVFQMUNYM0I0TmlFRXV2MndJVXFhV0F1TTRIeDBKKzNRUjR3TVRrNlcKRWJ5emRscVRVRDY5YWtUQ0JuNXJWL3Z2RElOSWdXTXFYSDA2UWtFajhnVExwVU0xUjFsVzhETFRLcUIzY3RTUgpsNzV0LzRGWVZHelg2Y0RQejVORGFldktvb0NJbWQ5VEs5Q1hSZ2lPYVhjQ1hFR0dZMzA3d0l6QmlRc2g1dGZ0ClNtTmVGSTRJWVg1WWU3aHVHVlpxOHBWOUdWeDJ2Z1JxNWdSMEY1OEVmMDNFMzRmdk54NWZobi85WTVqZGFQdTYKN1ZGajBNT1M4UkdyWVFpWmx0MlQ0TkYzQmFPMUpXVEZuc0JzQzR6NEI2c0dWaG9kTndCVnBaT3A3VHRyVWJUcwphN0lMMHlMUy9MbWlyQWhRUjJRb0x3b3lad1lpU0xIRkJreGVzeDFETlppYjArY2kzd2d1N2NWemRQYkVicWdyCjNKYzBpbUhzaTViaWFjaENRc3A1Tk5xL1VoZ1JaciszWmllQmFOQ1VlR09BTWZMR2ZseUFoT1lRM21VUWpFTWEKQ3pENW9xWWVqYU14eDd1R1VFUDh1dzJpSC9vZlA3OGtESUN5M2xtSit5eFptM255MEFFc1g0TEQyRW9PaW1oYwp5MGp6UXVsSnBiSXh4aGtNSEhHdDBoMEZwbk5rbHc5NVlKOEpEYjlWYzNwdjRSNkpyV3FlZkV1OXpoQzVvWHFYCmE2aUpLZE5mS21iT3d5cTlYTWJ6cGszQUpvOWVXMFZhSGdpMU1TL1owdjc4bTlxV1d1Q3V4bUxnNnBHYTBnWTMKRnQwYVRHSnFxQ004c1B1NHI2dEJBb0lCQVFEMlFYV1UvWU94Ti9YcTZ6VEx0Y0NTUC9zcU9HQThWanpwUnZ2TApiUDZFVnpPb29HRWFvSkFLTTFURitmRzVFK2dsQlljTUNxU2t5Y3EyZXV0aHM3MW1hUFgrdzJCVVBYOGZ5VmxFClFGUHQzQXlGV3d3bzFOaFZkelBDYi9SNVMwWVFUS1VNZytrSHhOM2M4Q0pmU2lZT0xRaHVrQ09JRHp6U0hqV2MKUjM4c3prS0IzaDlCTXEzMFF3TmhqS1dTRlVMaWloZkFrT1krYmlvQWxqTWgyRzVCb2RwWEVNSmxlNldCSTZmSgpaNG1wcjNDcmQvN0xEZUhIUzdUVG04UUsraGU0TW5PaHBFWEVjS2kyeHozMDV4Q3pRNnZFbmJHSWNKc2U0eDBHCk11c3VHVzMzaGMrb3NodjFtZHcxL3FFUkhoZitLMU5CQzZQZEh1Q0ZHMVQvbWFZVkFvSUJBUURha1Q3cmJtSEUKWlBqTHVNY0ROaVBpT2JER0Faa0pkWE9BVjlBQWtrTzJ5R1BUZTRLUkgyVThWc2dqekJjLzk2RlA2eW9HNVQxTgplbDlXWi9FcmZlbXM1NFdXeVpUOGpYWXFpeURBNG9Yekx4M3Bkb3ZVRUZnckRBZUg1WXl5Ni9mallBdDVqS1ZrCkw5aFh1cUNFZlBXZmxOaUJ0OE5pVXNCQUcrRXpsNVBzQ0xIQnlvM2x5OXUyTTFibEEvVkF1eHNuckhoVGpwWVcKNHQ3NXpDcjByTW5Ha3lPSDdvMWp6T1VJYU1CcDVid292bHdTb1pIWkRUSkdFcWhiSlRSYUJlQ2JLYzhBUFZHNQpMbm9VUnk0b3BISGFEcmEyWVRHeUN0bVhtK0h4WHkzeGhmZHFzUWFuQ0pVY0p5b3NwMFJvL2lxYWJpQ3dYWVUvCk9Cbm55clQxSlNIVkFvSUJBR3VlWmVXTCtWNmNwek5ZUVVWNWs4UVdoQXlLZ0x3OXIvYisxNUdxZTN5WW8zSGgKVFM2VzF2d3VQTEVjcjJBRDdDTXB6RUFkOHFBMXRBcVZvNEthUzM2VEJsYWxTZGJtM1VTbCtRWVQydG9MbmNrMQo1aFYrRjJFYWJCdGdWQVlpT0dkdEo0QlZzYVI4aTcwL2tMWDJNTFZuUnRVUzF3UmlMR0ZqWkdoODhuNUJVZDF4CmxsVW04ZERhN0lKWU5nK21qUWwxOGpWczNjS1E0SGhMSytOeHM1V3BSME5maHFWVktScEwyOHJ3SGNCemRKanIKSXdYWWRrQmp2STN4OS9ZWUgvK1d4T1B5WjY4VzBSUzM5RUt3TEtNN1FyajFkWjI4SUg2YUlKZ1I3cWZCNDBZVwpTNDljNzAwaFJaU3ZSL0swSlNZbUJ3ZFpMKzYxek1jL0Q2RjRvNVVDZ2dFQWJjUDk5bHlVQ3Y2dW1Ca3ZFU1RTCmRwMkVjcHlBeitoRlhsSTdhdDRKMWJUanRXVFUySzhNdDNYWncyaU8wSmc3VWhpSEhibG94UTFNN2ViN2psMEkKeXNYbktDZ0tnNTlEbGZBVFBldEZYRER3Yzd3T1V5ejJLb0E3RS91clludnhIU2F4L0pRdXg1Ympyb05TYzljUgp2OWdQdDIyaldUQzN6anB5S2VmWTZQUWcyWE14T2hQY1BxK2YxeG5heEd4ekljU1RGVnVKY3VyekVqNS80Q3NhCmxuaDBvcUtpTFZuTU9DSHJhQU54TUlFUldtWDhDaVovZGdPT3UxOSs0Q3NOZHI5VGJ3cGNqWVNTMkxZNnJ6eU8KMVBVSXU2VXFRUUVEOEFqZ09za1RHTFd2NE13UXpEZ2FNbTVVMXVJV0VDaDlHdHR0M1VUS1UwcUljQWswUWQwcApGUUtDQVFFQWhiMlphM2kyY2NlUVBzc2MvYVBqR2NVMjErVWZvbE04Vndvck5kNE93eFZSTEpUNXZwN1lzQ1pKCjdUSmJvUXRpNlZVRmZXUGNsb2JiWnB0b3lURktDcHRwNm9OejMweTFKK2VEaVhzR2lNZEo1Y3ZDVDFXWWhLSjQKWE1lMXhTZnZYVTlMcEZTVURscmczcm0xYVZXV01TU1Z0Q1RIODRBdDdQNUhBN0o3ZGVaM2ZjOVJxeUg3NkNYNQpiZVBHWC9SWEluVFN6T20zOHZUaXpYYnVuMjlxMjV0bW93WEczdEdOTHE3K0xNUlBBVCtQYWhyVnVyakRwb3d1ClpqSFRYMFM4N0VTa2QwYXRSMHd3YUM4R2IvUkgyUmhkNGphQzQ0WUlBNzJBS0xQdEdHaE0yemIwY3hRM2VpK0YKZXFiWGJEVEsybHE5LzVCL2RLRmNDNnJ1MGo3R3V3PT0KLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo="

	publicKeyEncodedRSA = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQ0lqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FnOEFNSUlDQ2dLQ0FnRUEwajkxdXN3RTdlcDJPWmt1N2pUOAp5dTlMUmtHMldVN0dXSFJ4TlQ0eXFZSmprWjNWcHJHVE8zWjJ3aFYySHU2S2NVSXY0YTYwY0VjZEtObWZEZDZ1CnYyZStJbXpsUXBHM1VwWWtLRVBWeVdpRU4vOS9LeVk4RW1FK2VKME1qdHJ2aTJpOWpJUEEwZmtZUnVJYTBRU3QKTUhSdk93ZXhSSkppZnJkeUhDa3hXNmNiVXZETm81ek9IUE1jVzE5L3ozQ1hKRHJGVnIwaHcvdmUzbEcyR2c1Wgo0T1k2eU11eHZxQVUza3RzdGx6akdnNjhHQnVMc0RzeWxoN0d1cWo0aTF4b2ZFTVhKMDlHZXdJQzZadmRKZ0k3CmY0YThYbzZLNUxzNys1RmpJWW9CQlNmVnNDdnNxN042MStGZGpjZ1VMWXN0SDBoOTFkVlZsWFIrb242S2JpUTQKNGtLYldXcnRqZnpxY3FnMWIxWDFEdm1WSEdYR0NLNW9rTU8wUk91ZG9DYWRTSjZ1UGhEamFKS0xHeE50RTBOcQo0V0JDRmZvMTlRdi9iUDl2dFVVTGJ5anNiZ1hmb21MMklVemVwOXkxU1JmNEhpeForV2tOdzc0dmZlSGN2a25WCmlidXN4UFljbHlneXcvb1hRcGtUZmlJMHFaZkpVVzZWbkFFNGNmSnA1WWlIWTExUGFmZjhIZG5FY0RGKzVyazYKZ2lETkxFK2lMZUJybUxuRlBHOXFiRHFpY1pPZWh5NThpcUwxWVFZa2pzaDg0dmFFemdhZ2IwNjFuTlBpeW1KNgpJR3owNGhuTmRuZjZqdlUycUs4ZDNlMGRnZDJZN0RPR1JVWHNKRVVTZkZaZW1QcElMMTZScTdPTG1ialFMV3hzCjVEOFQ1Nm5RWXpudklqVUlscE5JNUhrQ0F3RUFBUT09Ci0tLS0tRU5EIFBVQkxJQyBLRVktLS0tLQo="

	// Testing importing an existing key that's been added to TUF already.
	// Generated with:
	// openssl ecparam -name prime256v1 -genkey -noout -out privkey.pem
	// openssl ec -in privkey.pem -pubout -out pubkey.pem
	//nolint: gosec
	existingEncryptedPrivateKey = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIM6pOLxVCBLPNcwsA7BOOb9k4c0q//YjX2eSzGeLBru6oAoGCCqGSM49
AwEHoUQDQgAEIvSnDm70zQ5+ezI0jetTGrPIOhetyv0ENmll0DjhsRKFWAX8zT38
cXaAnFOJsC5M011+x6v+IMNkY/1jrWaHfw==
-----END EC PRIVATE KEY-----`
)

// testConfig wraps the private,public, and config into a single struct
// so that we can test with different keys.
type testConfig struct {
	private string
	public  string
}

var testConfigs = map[string]testConfig{
	"rsa": {
		private: privateKeyEncodedRSA,
		public:  publicKeyEncodedRSA,
	},
	"ecdsa": {
		private: privateKeyEncodedECDSA,
		public:  publicKeyEncodedECDSA,
	},
}

func TestUnmarshal(t *testing.T) {
	for k, v := range testConfigs {
		t.Logf("unmarshaling with %s", k)
		in, err := createBaseConfig(t, v)
		if err != nil {
			t.Fatalf("failed to createBaseConfig: %v", err)
		}
		config, err := Unmarshal(context.Background(), in)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		t.Logf("Got: %s", config.String())
		if len(config.FulcioCerts) != 1 || !bytes.Equal(config.FulcioCerts[0], []byte(existingRootCert)) {
			t.Errorf("Fulciosecrets differ")
		}
	}
}

func TestRoundTrip(t *testing.T) {
	privateKeyECDSA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate Private Key: %v", err)
	}
	privateKeyRSA, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		t.Fatalf("Failed to generate Private Key: %v", err)
	}
	privateKeyEC, _, err := ParseExistingPrivateKey([]byte(existingEncryptedPrivateKey))
	if err != nil {
		t.Fatalf("Failed to parse encrypted Private Key: %v", err)
	}
	for k, v := range map[string]crypto.PrivateKey{"rsa": privateKeyRSA, "ecdsa": privateKeyECDSA, "ec": privateKeyEC} {
		t.Logf("testing with %s", k)
		var ok bool
		var signer crypto.Signer
		if signer, ok = v.(crypto.Signer); !ok {
			t.Errorf("failed to convert to Signer")
		}
		configIn := &Config{
			PrivKey: v,
			PubKey:  signer.Public(),
		}
		if err := configIn.AddFulcioRoot(context.Background(), []byte(existingRootCert)); err != nil {
			t.Logf("Failed to add fulcio root: %v", err)
		}

		marshaledConfig, err := configIn.MarshalConfig()
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}
		configOut, err := Unmarshal(context.Background(), marshaledConfig)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		if !reflect.DeepEqual(configIn, configOut) {
			t.Errorf("Things differ=%s", cmp.Diff(configIn, configOut, cmpopts.IgnoreUnexported(Config{})))
		}
	}
}

func TestAddNewFulcioAndRemoveOld(t *testing.T) {
	ctx := context.TODO()
	for k, v := range testConfigs {
		t.Logf("testing with %s", k)
		in, err := createBaseConfig(t, v)
		if err != nil {
			t.Fatalf("failed to createBaseConfig: %v", err)
		}
		config, err := Unmarshal(ctx, in)
		if err != nil {
			t.Fatalf("Failed to unmarshal %s: %v", k, err)
		}

		newFulcioCert, err := createTestCert(t)
		if err != nil {
			t.Fatalf("Failed to create a test certificate: %v", err)
		}
		if err := config.AddFulcioRoot(ctx, newFulcioCert); err != nil {
			t.Fatalf("Failed to add fulcio root: %v", err)
		}
		marshaled, err := config.MarshalConfig()
		if err != nil {
			t.Fatalf("Failed to MarshalConfig: %v", err)
		}

		// Now test that we have configuration that trusts both Fulcio roots
		// simultaneously while one is being spun down.
		expected := [][]byte{}
		expected = append(expected, []byte(existingRootCert), newFulcioCert)
		validateFulcioEntries(ctx, marshaled, expected, t)

		newConfig, err := Unmarshal(ctx, marshaled)
		if err != nil {
			t.Fatalf("Failed to unmarshal new configuration before removal: %v", err)
		}
		if len(newConfig.FulcioCerts) != 2 {
			t.Fatalf("Unexpected number of FulcioCerts, got %d", len(newConfig.FulcioCerts))
		}

		// Now for our next trick, pretend we're rotating, so take out the
		// existing entry from the trusted certs.
		if err := newConfig.RemoveFulcioRoot(ctx, []byte(existingRootCert)); err != nil {
			t.Fatalf("Failed to remove fulcio root: %v", err)
		}
		marshaledNew, err := newConfig.MarshalConfig()
		if err != nil {
			t.Fatalf("Failed to marshal new configuration after removal: %v", err)
		}

		// Now test that we have configuration that trusts only the new Fulcio
		// root, simulating that the old one has been spun down.
		expected = make([][]byte, 0)
		expected = append(expected, newFulcioCert)
		validateFulcioEntries(ctx, marshaledNew, expected, t)
	}
}

func createBaseConfig(t *testing.T, tc testConfig) (map[string][]byte, error) {
	t.Helper()
	private, err := b64.StdEncoding.DecodeString(tc.private)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode privateKeyEncoded: %w", err)
	}
	public, err := b64.StdEncoding.DecodeString(tc.public)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode publicKeyEncoded: %w", err)
	}
	return map[string][]byte{
		"private": private,
		"public":  public,
		"fulcio":  []byte(existingRootCert),
	}, nil
}

func createTestCert(_ *testing.T) ([]byte, error) {
	// Generate x509 Root CA
	rootCA, rootKey, err := testutils.GenerateRootCa()
	if err != nil {
		return nil, fmt.Errorf("GenerateKey failed: %w", err)
	}
	// Extract public component.
	pub := rootKey.Public()

	derBytes, err := x509.CreateCertificate(rand.Reader, rootCA, rootCA, pub, rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	return pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: derBytes},
	), nil
}

// validateFulcioEntries will take in a marshalled config and validate
// that it has only the fulcioCerts specified as well as matching number
// of fulcio entries in both the CTLog configuration as well as in the
// passed in map (that gets mounted as secret).
func validateFulcioEntries(_ context.Context, config map[string][]byte, fulcioCerts [][]byte, t *testing.T) {
	t.Helper()

	foundRoots, ok := config["fulcio"]
	if !ok {
		t.Errorf("Failed to find a PEM for entry")
	}
	expectedCerts := make([]byte, 0)
	for _, f := range fulcioCerts {
		expectedCerts = append(expectedCerts, f...)
	}
	if !bytes.Equal(foundRoots, expectedCerts) {
		t.Errorf("mismatched PEM entries")
	}
}
