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
	"strings"
	"testing"

	"github.com/google/certificate-transparency-go/trillian/ctfe/configpb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sigstore/rekor/pkg/pki/x509/testutils"
	"google.golang.org/protobuf/encoding/prototext"
)

// Just a test Root Cert from a Fulcio instance spun up using Scaffolding.
const (
	existingRootCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
`

	// testConfigECDSA contains above cert in it as well as privateKeyEncoded and
	// publicKeyEncoded.
	testConfigECDSA = "YmFja2VuZHM6e2JhY2tlbmQ6e25hbWU6InRyaWxsaWFuIn19ICBsb2dfY29uZmlnczp7Y29uZmlnOntsb2dfaWQ6MjAyMiAgcHJlZml4OiIyMDIyLWN0bG9nIiAgcm9vdHNfcGVtX2ZpbGU6Ii9jdGZlLWtleXMvZnVsY2lvLTAiICBwcml2YXRlX2tleTp7W3R5cGUuZ29vZ2xlYXBpcy5jb20va2V5c3BiLlBFTUtleUZpbGVdOntwYXRoOiIvY3RmZS1rZXlzL3ByaXZrZXkucGVtIiAgcGFzc3dvcmQ6Im15dGVzdHBhc3N3b3JkIn19ICBwdWJsaWNfa2V5OntkZXI6IjBZMFx4MTNceDA2XHgwNypceDg2SFx4Y2U9XHgwMlx4MDFceDA2XHgwOCpceDg2SFx4Y2U9XHgwM1x4MDFceDA3XHgwM0JceDAwXHgwNNWwXHhlM1x4YTZYXHhjZS9ceGE1XHg5NFx4ZjZceGM2Plx4ODJceGJje1x4ZGVceGYwfG0rXHhkMVx4Y2U7XHg4NVx4YmZceGYyXHhmOFx4OTRceGYwfVx4ZDlceDFkPlx4N2ZKKFx4YzY+cVx4OGZceGM4XHgwZVx4YTJdXHgxNFx4ODhceGM4XHhkNX7Du2ZzXHhlZVx4OTlceDFicVx4MGVgR1x4ZWZceGUyQlx4ZjQifSAgZXh0X2tleV91c2FnZXM6IkNvZGVTaWduaW5nIiAgbG9nX2JhY2tlbmRfbmFtZToidHJpbGxpYW4ifX0="

	// ECDSA private key
	privateKeyEncodedECDSA = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tClByb2MtVHlwZTogNCxFTkNSWVBURUQKREVLLUluZm86IEFFUy0yNTYtQ0JDLDJiNDU2MGUyY2RlMGE3ZWM0NjZlMzkzYWRmYmE0Y2I0CiAgICAgICAKVUk4d2lUbXhNajhKWXVHSUFEMnpKVjRmQjZHUE9wUGhxSldYdlR3RWFucHBzTXN3UUFCaVZ5NWdkSi9BNThQVAo0ZTFFSDM4Y3Z3YTBMQjQ2SHBoZW9vWCtJM2RHdHlzRUpFR0d3QXMwYUhkU25aeVV3TnRpalRUQkZJcWxzd3pKCnI2WmJ4dmlxZVRmRm80ZUtEMGorRjlja2R3d2dGT2YzRHdaUUMrNEN1cVNqczdaZkFKZEF6Lys0c2JRd1ZzQUIKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo="

	// ECDSA public key
	publicKeyEncodedECDSA = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFT3Y1bzVXV0tZaVVSODdzNGZpMEpKbU1EUVV2cQpSck1mNGRlQnpzV3BCWVdVK1Y4TXVDMkh6aTFOTHI4czRlQ0J5dWVDZmFQWFN4STgzUkowamEwbnd3PT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="

	// This is for RSA, since previously deployed CTLog used RSA.
	testConfigRSA = "YmFja2VuZHM6e2JhY2tlbmQ6e25hbWU6InRyaWxsaWFuIiBiYWNrZW5kX3NwZWM6ImxvZy1zZXJ2ZXIudHJpbGxpYW4tc3lzdGVtLnN2Yzo4MCJ9fSBsb2dfY29uZmlnczp7Y29uZmlnOntsb2dfaWQ6ODMxMzUyNzQxMDgyOTkwNTY3OSBwcmVmaXg6InNpZ3N0b3Jlc2NhZmZvbGRpbmciIHJvb3RzX3BlbV9maWxlOiIvY3RmZS1rZXlzL3Jvb3RzLnBlbSIgcHJpdmF0ZV9rZXk6e1t0eXBlLmdvb2dsZWFwaXMuY29tL2tleXNwYi5QRU1LZXlGaWxlXTp7cGF0aDoiL2N0ZmUta2V5cy9wcml2a2V5LnBlbSIgcGFzc3dvcmQ6InRlc3QifX0gcHVibGljX2tleTp7ZGVyOiIwXHg4Mlx4MDJcIjBcclx4MDZcdCpceDg2SFx4ODZceGY3XHJceDAxXHgwMVx4MDFceDA1XHgwMFx4MDNceDgyXHgwMlx4MGZceDAwMFx4ODJceDAyXG5ceDAyXHg4Mlx4MDJceDAxXHgwMFx4YjlceGEzSVx4YTVceGI4XHgxNTlceGU0Qlx4ODdceGMzWlx4MTZceDExXHgwMHPknY1ceGVmXHhiYzlkXHg4YVx4YjZTXHg5Zlx4YThMXHgxMNWGXHgwNVx4MGJceGU1XHgwY01ceGNlMlx4YjZceGYwXHg4MFx4OTVceDAxd1x4YTBA0rdGXHg4NipceDgxRFx4YWU3XHhmZFx4ZDlrMlx4YmNzflx4ZTF5XHhkOFx4MTZceGY2XHRceDEyXHLKm1xuXHJceDFhXHg5N1x4ZTZceGIyXHhlYVx4YzBceGZhXHhiY2VceGE1cFx4ODhceDk3XHg4YTdceGZmXHhmMVx4Y2V2XHgxY1x4ZGZcbsiwLVx4ZGNceGQ0e1x4Zjl+XHgxMCRceDk2XHhiYzggXHhlMlx4MWVceGMyXHhkMlx4ZjNceGM3aVx4MGUtXHg4ZVx4YjZceDg0Llx4MDVceDE3JVx4ZTRceGExXHgwZlx4Y2POjVVWOVx4MThEJVx4YTdceDgzT1wielx4YTdceGU3ZHRceGExRExceGFjXHhlN3pybFx4MTBceGQ3QFx4OWVdXHhmMGRceGQxUl5fOVx4ZmRceGE3PzQgXHhmN1x4MTNcXFx4Y2ZceGU5XHhjN2xceDAzKVx4ZTljXHhkYlx4MDE4MVx4OTl9XHhlZjJceDhmRVNIXHhmZmdceGY4XHhjYklceGI5XHhiOVx4ODNceGEyXHhhNlx4ZDBceDAxY1x4ODc/c1x4MDNceGZiXHg4N1x4ZTlIXHhkYXlceDAzXHhmM2RdXHhiYXtceDgzXHgxY1x4YjdcXFx4YTZceDA2PVx4MTNceGU0XHhlYlx4ZDNceGRlXHgxMVx4YTdWX2tQXHg4Ylx4YzBceDhkXHhmY1x4ZmFnXHhiOFx4YzBmS1x4YjQtYVx4Y2RTXHhlY25ceDhhXHg4MUxdXHgwNFx4MDBceGFmXHhlMVnUl1x4MGZiIVx4MDNceGJhOXYlXHgwY1x4ODNceGYxXHgxOVx4YWM6XHgwYnRceGZjXHg4NlFceGIyXHhjY1x4ZjBceGJiMVx4ZWVceGFiXHhlMERceDAzXHg5Yy1ceGRkalx4YTRceDg4MllQVFx4OTBceDEyXHg4Y0R5dFx4Y2RvcDVceDFmeVx4ZmR2XHhjN1x4MTZceGIwXHgwNDFccnRDXHgxOTckXHgxMFx4ZDJceGUxXHgxZFx4OTBFXHgxNSnuqYtceGNjXHhlZDp1XHhhMFx4ZTRceDEwXHhkNGJZXHhmY1x4MDTDsybOgVFceGRkRlBFXHhmMWs6Wlx4YjZceDlibWpceDE1XHhkN1dceGM1XHhkZVx4ZTdBXHhmMlx4ODdceGRiXHgxNVx4ZTBAXHg4Zlx0XHg4M9mWXHg4MEVJXHgxZFx4YTVceGFjXHg5Mlx4Y2Jmelx4ODJceDg1M3dceDkzXHg4MVx4ZWVceGM0a1x4YjZceGJlWWxceDk0XHgxYTpgXHhlNFx4ZjJceDBjXHhmMFx4YTAjXHg3Zlx4YmEvWlx4ZDA6fVx4ZTNceDAyXHgwYlVbXHhmNi1ceGQzUlx4OWRceDBi4pGE2ZJceDk3XHg5Y1RceDdmXHhmMVhceGIw66yvXHgxOVx4OGNceDg3XHhmNlx4ZTBceDFhTV9aZ9yXXHhmMng9XHhhMVJsXHhhYlx4OWRiXHhmMVx4ZjFnPVZceDhmaVx4ZWNceDdmXHhlM1x4ZjhceDFmXHhkYlx4MWJiXHhlMGtceDkxXHhkN1x4YzdeXHgwMFx4MTQ0XHhkM1dceGViXHhhZFVceGQ1XHhkZlx4MDJceDAzXHgwMVx4MDBceDAxIn0gZXh0X2tleV91c2FnZXM6IkNvZGVTaWduaW5nIiBsb2dfYmFja2VuZF9uYW1lOiJ0cmlsbGlhbiJ9fQ=="

	privateKeyEncodedRSA = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpQcm9jLVR5cGU6IDQsRU5DUllQVEVECkRFSy1JbmZvOiBBRVMtMjU2LUNCQyw3NWUxNTkxNzQ0NTc4MjMwNGUzYjY1NGQ5NjhjY2M4MAoKV1pPQ1QrQXlaUmlaaFpDdXMveGxuR2dFbzNwTk1GRSsra0YvWVdBZUxMQjhmclNuL2NlL3VjbURuOURGQ01VZApORlNhSks1YzNvWEJCckt0Uk1sQ0I2S2RGblJucHNpVHUzbU1sVzVPdzRNTVh0L3JJaEFXbDFDaUFYUkdqL0NWClg4clRvQldpOFN4dXh3aWgrOHlrY0VpaVg3Ti9aWkNYOUppbjFQeTc0QUczWHBPT28rbFhwKzRTN1BwQmlZbzAKU0pzaUZ4Mlk0LzF4RXBWMEVWdmZobmN1R0k1R0ROcm0wUnBBNnNraGRSbU5iMW1HYkR5ZXdnMndPTTJTRHRGQwpSWEE5aFAxV1czUWx0VGhXRml2VTU0SngrYktMc3Fnem9JMzNZRmRFdnRPNmNxWCtoOVprN1pORmxaMDNaREk4Ck5RdzEyT3Z3VnpEeE5XdmFYVFhIMEpJc2tUSTE5cjFCTnB6aW1xdWg4ZWRYSTFuT2ppbUM5VjlRQTF0TVNmWmkKVmM2RW9VSG55N0xNVXkydG1yN3R2M2pLRWJHT09nclNRcXhJejAxcjFtV0dpREU2YkNDeFFueUhOUHExQmlIRQp1WTR3K25iU2V5UDhVc3h6YjlVNkRSd2IxVzZkMjlmbGNsdFp1TFlqdEhRL1JwRUdxbWRNc1RmRU1wRUVTNU9jClJPVmtsQlpQM0NHN3I4NGN0aVBMUGpvZnk0aG4rai9SeTBtT2tzcFcyVjNlQ2FvdGQwU0lQZFhxT3h6K2p3U1kKaDRBelg1VHdMSlg2UDlSaVdVZ2xQUWZKNjhCclpOT1Ywc3IwaEIwc1NXY25mSWorWWxSSzMvUXJTZGdhellRRQo0ZHBrK0hDUUE4bkdwN1M1Uks4ZGdxek1QYS96Z1AvR1dnN0t5K0dVWFB3cXRhalBFd1ZVWFJPNGViWUJCQ1RwClFHYnRSSmdRRjFzSmtqN1F0d0J4NzVoM25ZSjlWdEhiMWR2d2FKL09mWklhSklKQkRROVlyRGtqMjVmdDdtWlEKZVlGN1c5NlhCU0xHc2ZhdzlDMXhNRXZVY081UGtkS3ArR3pvMFhUaXhNb1U1Q0h6Yk0rQnFqMFZycGpNV29XbQphbHZpYVc4RlNYQkZQZUNoNFIrOXhwN1Q3ZWl6OU9uRFpKRVdnR1B1YXZyN29XL0t6blE1RS9SVlJtRllaZVY2CkluRXlmUVlRVE5QMnVBWjdibFRCeEc0VlhWdjA4ZUhWVHJ4YkVBcmE4VXJrZkQ1Nm02U3M3YWsrYU1mdG0vSnkKZHBxbTJ5YWlpSDd1SmRiZ1hyNTBnNEFDUThtZlE1QjNpbk1Ea0NFZ2RyQTRTQXg1YXNaQjJ0V2l1VC9SZFVSLworMUpXbjNKdXBEL2dhWU5CTVBTRzhjL0hKa0xmeE5UdzZVaHBBTlg5TkErTlE1UVdCUTVaaWNhbUNLQWJUczEvCjhUUlJlbnBLdUdhZXVsazhneVNOTm5xa0plZUNlZ1c5RGR1d3BZcUpjVkJ3L3lrY3BDc0hleVVZSTFOZkd0dCsKcTJ0Z2h0WGhaSGpFV1ZhcWVIb0JOTHlxZ0NET0l6U1QyTnFSeC9yYXhXckl1K0JwMTJTazNpQm5pc1Y0cE02NQprMnFaTDVhY2FDb3lIWTlSWStKSThYdHBzcHVjclViZnp6K0F3ZVZpdkcwN0hkOWRnV0dMRHRwMDJ4VGFMb09pCnp1NnV1dU9heUtZaUI4N2RBYlJlZUY0RVNrTlZOM3k4c1hIS3lnRlFvN2pqRExWVnBwRVVYWC8vN081VU9aZ0wKMWtWcVJ4K2hLeTQvTnVqQUVReWJubnMzRlpIMHBDMDQvcnAwS2xBeHlmRzBRNWJvTWdBeUR0VGlyUFBzK2lwTQpveDh1aWdlQlFaTmZyWW41TVA2UWVUSWY0QWx4NWNzSktxb1Nzb2dZclljbWhoSkhkc1Q4QUpidlpXSUo4L01JCjRFKzJ6UEZSNUlOYzNGbjVoVFpnRzNMQjh4N0ErTHlCbEdNR2owdW9melVzdnZMNnpxeEtqZ3F1Qm5DbTNmTHYKSjFnaDFYbkUyeENVekZhSlpQOVVNU1N2bmVmci92TzBFMjFxL0NlSGRUNWZsaUl4UjBZQ0t5MENvd3ZIeUdyYwpmc2JWWS92dGhIcUxLYmx0Vkh0bndPOExFTmhWZmVweGhFUy9sQUZrWmgrbmZFYjVsUnRZb2hZSW9RUkFOR1A0CmhCS1BhWldua0kwbFl5TmJNU1h3d3U0R2lScFdUUjhUYW84WDlXSWlJdmgyc3hHd0NleTBPSGZCVGtoYnR3Y2sKQzlaT0pERW9SNXBlOGZXSitzWXBia1laYjd3TzhSVEMyNlBGZTRQdEtKRFNGWXlOMzM2T1ZVdzM2RkZmVzR0QQpvcGtBdkRVbDdXVEZ1TlB4RVZ3SXZQSnN2ZDdnaG9Kdm1MYm4xQldQTS8wY0lobkQ0YkdrbjBsVURTTUFjUXIvCkV2R0h4Z2xpeU4wdktnOWU5SE9VNkVOYVdMaTRzemhwdE05RzF6UnBic01CV05zRW5TTEEwL3BaS01TOXdGdk8KL1N2VEVFc3dlM2xKWjV3WFc2R3lUdURFMzQ4Z011UFk4RmpCajZjQVo3RUJLTmYrWG1TY3VQTHYxVzd3Nm52cApKTGtQRS8wQmswdEZWRndlZUlERHJOTEg4Z0dseTY3MHk5cUxQSi8rMUhwdXpwR2tqc3RwWEs2QkRqWXUzeEFlCkhsd3E2RDNmRTMrZ0VkcW5RUmhZeHRacWxqaGIydFIxYUErZndhcWVBT2dNOG43RkNaY0gvK0ZBakdhRis3YjEKQ0RIdjA0cktKdVFGZjZTKzNzQktaVW9aVllJakxidE9VWko4c2QvZEZaQ01mNGhnN3RiaXNQeVFxMjQ2MUI3Wgp1SnFidlozdHhiT0lpd3k0cklCT2VtTnJaR3ArYmMzT3FuOHZQaEtpM3c2aDd4M2lvUzBxS250bStMbG11MXBqCnZOZnQwNmFZYklGcUhkY3ZqQ1AxajZNemY1Rm9TMGhmVnlpRmltOVFUOVpGeDl4bDNGeHBkK2VsYkxYY09pM0YKU2dISWE5SUdYQXNsSmo5dE5zdC9GaHBxeFdQbmt5c3dNTjRCQkJ2SDJNZU5odWpVUGdWblp5bEVodU1jQTBrcgpzdWMrNmliMEdRYUhRSW1pOHpmQ1FyQUVXMzZ2WWRxK1M0ZjBOeEZVNFZkclFzd2tpYlJhSytBTkZGY1ZKUzFJClcxWFdoU0FKV2VPUjJONmxJVFNqZVNDbXc5bnlXb1prZXBvSEkrcTlDZ0cvV09qRy9ZUkdjZUJNSFZQbk1zNDYKanA1NitvQkdXSUVpK3dvRU51UFV0aDNlZnZNT2dGTlBGZWh1QUFUUHBOeGtaMlBheHpRVmp6NXJGR09tNmJtZgoyWExIQVZxcTFjYVhEY1RidGxoSWh0Q3A5cmlGcXIzc2R6YlFxWThCWUsyQjdyQ0JHbXFjZld0akgvWUZadkNrCnFWNWpoOHQ2MFp2Z2F1bU15Y2h2NGNVaFRWMFJzZ1BteE9GMzdUenY1T0d3OVBKeS9sdFphNncveFFZQWVMaHQKRnVWN0I0WFJvdERyYklvZkNNM1ZObXdXTnN4R29LNWY1LzV6bVBEQ1JQNjZDNkkwbWVLNjZXb3prY0N2NTRMcQpJZDJaZTN5aUY2bjE0K05xZUZMWGVsdnRvay9RSWdiTEd3ME9XVEQyaFJtZGVYMjhUMEVMMW5kZ0ZUYU0xV3NlCkVJdXQxWXNLWXk1Vml2bDg1V0JiZEsvKzZuMjVIa2l3SGV5bHRsOWZ1cFEwSlcyM01yc1I2RWwybU1qQ0FFTEwKQ0l4TjdrOGFRTk92SndmV25LWjQ0U3BIalFPUXdtTTJySlVpZzBhZURUMWNMck9sVDNSVndUeG5DK00rN2V6SwpTZElza0ZZR0ZXdW12NlBZSVZBMy9MOE16T3dWeGs1WWwzcnpJaVh4UGlrdU1FeEtqNlRsNU8rQjBXQ2c0UVVUCjFGdk1zZksxNUwrRjdaeExuVi96WTVmQ2VBUEY2dXZDYjJ4VFBBeGZwN0VxK0tsSEdybzBWb1UwSGRSNFJLR2YKZlg0TytkZ3NNUHB1K1lQWTBWVGZTVjdVN2dWdklPcHhzc2lQbXQwdmRLSjJLK04xWUV5TmdKVlBCNUtyVXZveQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo="

	publicKeyEncodedRSA = "LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1JSUNDZ0tDQWdFQXVhTkpwYmdWT2VSQ2g4TmFGaEVBYytTZGplKzhPV1NLdGxPZnFFd1ExWVlGQytVTVRjNHkKdHZDQWxRRjNvRURTdDBhR0tvRkVyamY5MldzeXZITis0WG5ZRnZZSkVnM0ttd29OR3BmbXN1ckErcnhscFhDSQpsNG8zLy9IT2RoemZDc2l3TGR6VWUvbCtFQ1NXdkRnZzRoN0MwdlBIYVE0dGpyYUVMZ1VYSmVTaEQ4ek9qVlZXCk9SaEVKYWVEVHlKNnArZGtkS0ZFVEt6bmVuSnNFTmRBbmwzd1pORlNYbDg1L2FjL05DRDNFMXpQNmNkc0F5bnAKWTlzQk9ER1pmZTh5ajBWVFNQOW4rTXRKdWJtRG9xYlFBV09IUDNNRCs0ZnBTTnA1QS9Oa1hicDdneHkzWEtZRwpQUlBrNjlQZUVhZFdYMnRRaThDTi9QcG51TUJtUzdRdFljMVQ3RzZLZ1V4ZEJBQ3Y0Vm5VbHc5aUlRTzZPWFlsCkRJUHhHYXc2QzNUOGhsR3l6UEM3TWU2cjRFUURuQzNkYXFTSU1sbFFWSkFTakVSNWRNMXZjRFVmZWYxMnh4YXcKQkRFTmRFTVpOeVFRMHVFZGtFVVZLZTZwaTh6dE9uV2c1QkRVWWxuOEJNT3pKczZCVWQxR1VFWHhhenBhdHB0dAphaFhYVjhYZTUwSHloOXNWNEVDUENZUFpsb0JGU1IybHJKTExabnFDaFROM2s0SHV4R3Uydmxsc2xCbzZZT1R5CkRQQ2dJMys2TDFyUU9uM2pBZ3RWVy9ZdDAxS2RDK0tSaE5tU2w1eFVmL0ZZc091c3J4bU1oL2JnR2sxZldtZmMKbC9KNFBhRlNiS3VkWXZIeFp6MVdqMm5zZitQNEg5c2JZdUJya2RmSFhnQVVOTk5YNjYxVjFkOENBd0VBQVE9PQotLS0tLUVORCBSU0EgUFVCTElDIEtFWS0tLS0tCg=="

	// Testing importing an existing key that's been added to TUF already.
	// Generated with:
	// openssl ecparam -name prime256v1 -genkey -noout -out privkey.pem
	// openssl ec -in privkey.pem -pubout -out pubkey.pem
	// openssl ec -in privkey.pem -out privatekey_encrypted.pem -aes256
	// And encrypted with this supersecretpassword
	existingEncryptedPrivateKeyPassword = "supersecretpassword"
	//nolint: gosec
	existingEncryptedPrivateKey = `
-----BEGIN EC PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,3C33CA88DF439D434ABDB2DD03491BEC

A9UPVwTxy82/vDcG9q/e5SDKYokAGYvMyS5KD9rfyS5RGGQDdpkQPK0q6v9AFJbn
VCphFSJvnjFAR90XgF2EK+fVpX2GQjFEPhODVzAmqjawZHfTeGeMU5cJ+nNW+O6A
71ay3pGMAEQAvrzEErTLzCsBf2HZV1ioeFZVwHysvAA=
-----END EC PRIVATE KEY-----`
)

// testConfig wraps the private,public, and config into a single struct
// so that we can test with different keys.
type testConfig struct {
	private string
	public  string
	config  string
}

var testConfigs = map[string]testConfig{
	"rsa": {
		private: privateKeyEncodedRSA,
		public:  publicKeyEncodedRSA,
		config:  testConfigRSA},
	"ecdsa": {
		private: privateKeyEncodedECDSA,
		public:  publicKeyEncodedECDSA,
		config:  testConfigECDSA},
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
	privateKeyEC, _, err := DecryptExistingPrivateKey([]byte(existingEncryptedPrivateKey), existingEncryptedPrivateKeyPassword)
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
			PrivKey:         v,
			PrivKeyPassword: "mytestpassword",
			PubKey:          signer.Public(),
			LogID:           2022,
			LogPrefix:       "2022-ctlog",
		}
		if err := configIn.AddFulcioRoot(context.Background(), []byte(existingRootCert)); err != nil {
			t.Logf("Failed to add fulcio root: %v", err)
		}

		marshaledConfig, err := configIn.MarshalConfig(context.Background())
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
		marshaled, err := config.MarshalConfig(context.Background())
		if err != nil {
			t.Fatalf("Failed to MarshalConfig: %v", err)
		}

		// Now test that we have configuration that trusts both Fulcio roots
		// simulating while one is being spun down.
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
		marshaledNew, err := newConfig.MarshalConfig(context.Background())
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
	c, err := b64.StdEncoding.DecodeString(tc.config)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode testConfig: %w", err)
	}
	private, err := b64.StdEncoding.DecodeString(tc.private)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode privateKeyEncoded: %w", err)
	}
	public, err := b64.StdEncoding.DecodeString(tc.public)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode publicKeyEncoded: %w", err)
	}
	return map[string][]byte{
		"config":  c,
		"private": private,
		"public":  public,
		"rootca":  []byte(existingRootCert),
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
// of fulcio-%d entries in both the CTLog configuration as well as in the
// passed in map (that gets mounted as secret).
func validateFulcioEntries(_ context.Context, config map[string][]byte, fulcioCerts [][]byte, t *testing.T) {
	t.Helper()
	// This keeps track of if we've seen a file entry in the CTLog config
	// for fulcio-%d entry. There should be one for each fulcioCerts
	foundFile := make(map[string]bool, len(fulcioCerts))
	for i := range fulcioCerts {
		foundFile[fmt.Sprintf("%sfulcio-%d", rootsPemFileDir, i)] = false
	}
	foundPEM := make([]bool, len(fulcioCerts))
	PEMEntriesFound := 0

	// First make sure we have all the PEMs that we expect in the map.
	for k, v := range config {
		if strings.HasPrefix(k, "fulcio-") {
			for i, fulcioCert := range fulcioCerts {
				if bytes.Equal(v, fulcioCert) {
					foundPEM[i] = true
				}
			}
			PEMEntriesFound++
		}
	}

	if PEMEntriesFound != len(fulcioCerts) {
		t.Errorf("Unexpected number of PEM entries, want: %d got %d", len(fulcioCerts), PEMEntriesFound)
	}
	for i, found := range foundPEM {
		if !found {
			t.Errorf("Failed to find a PEM for entry %d", i)
		}
	}

	// Then validate that for each of those there's an entry in the CTLog
	// config file.
	// Then check the log config that it tells CTLog to trust these two certs
	// above
	multiConfig := configpb.LogMultiConfig{}
	if err := prototext.Unmarshal(config[ConfigKey], &multiConfig); err != nil {
		t.Fatalf("failed to unmarshal ctlog proto: %v", err)
	}

	trustedCerts := multiConfig.GetLogConfigs().Config[0].RootsPemFile
	if len(trustedCerts) != len(fulcioCerts) {
		t.Fatalf("Unexpected number of file entries, want: %d got %d", len(fulcioCerts), len(trustedCerts))
	}
	for _, fileName := range trustedCerts {
		if strings.HasPrefix(fileName, rootsPemFileDir) {
			foundFile[fileName] = true
		}
	}
	for fileName, found := range foundFile {
		if !found {
			t.Errorf("Failed to find a PEM for entry %s", fileName)
		}
	}
}

func TestDecrypteExistingPrivateKey(t *testing.T) {
	priv, pub, err := DecryptExistingPrivateKey([]byte(existingEncryptedPrivateKey), existingEncryptedPrivateKeyPassword)
	if err != nil {
		t.Fatalf("Failed to decrypt existing private key %v", err)
	}
	if priv == nil {
		t.Fatalf("got back a nil private key")
	}
	if pub == nil {
		t.Fatalf("got back a nil public key")
	}
}

func TestDedupeUnmarshaling(t *testing.T) {
	for k, v := range testConfigs {
		t.Logf("testing with: %s", k)
		cm, err := createBaseConfig(t, v)
		if err != nil {
			t.Fatalf("Failed to create base config: %v", err)
		}
		// Override the legacy rootca entry with our own for ease of testing. It
		// doesn't really matter what it is for this test.
		cm["fulcio-0"] = []byte("this is a test cert")
		cm["fulcio-1"] = []byte("this is a test cert")
		cm["fulcio-99"] = []byte("this is a different test cert")
		config, err := Unmarshal(context.Background(), cm)
		if err != nil {
			t.Fatalf("failed to Unmarshal: %v", err)
		}
		// We should have original root cert, 0&1 deduped into one and fulcio-99
		if len(config.FulcioCerts) != 3 {
			t.Errorf("wanted 3 fulcio certs, got: %d", len(config.FulcioCerts))
		}
		checkContains(t, config.FulcioCerts, []byte("this is a test cert"))
		checkContains(t, config.FulcioCerts, []byte("this is a different test cert"))
		checkContains(t, config.FulcioCerts, cm["rootca"])
	}
}

func checkContains(t *testing.T, fulcioCerts [][]byte, cert []byte) {
	t.Helper()
	for i := range fulcioCerts {
		if bytes.Equal(fulcioCerts[i], cert) {
			return
		}
	}
	t.Errorf("did not find %s in fulcioCerts", string(cert))
}
