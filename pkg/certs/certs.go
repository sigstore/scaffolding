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
	"fmt"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// SplitCertChain splits a single cert chain file (in PEM format) into multiple
// files that we can then construct multiple TUF targets for each of them.
// This is mainly for TSA certchain, where we want to construct easily
// identifiable (by name) targets. For TSA the form is like:
// tsa_root.crt.pem
// tsa_intermediate_0.crt.pem
// <optional> tsa_intermediate_1.crt.pem
// <optional>...
// <optional> tsa_intermediate_n.crt.pem
// tsa_leaf.crt.pem
// The assumption is that the first entry is the `Leaf` followed by 0 or more
// intermediates and the last entry is the `root`.
func SplitCertChain(chain []byte, prefix string) (map[string][]byte, error) {
	ret := make(map[string][]byte, 3) // we asssume there's 3, no harm if less.

	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(chain)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling certificates: %w", err)
	}
	if len(certs) < 2 {
		// Need at least a root and leaf
		return nil, fmt.Errorf("cert chain must contain at least root and leaf, but got only %d certs", len(certs))
	}
	// handle leaf
	leaf, err := cryptoutils.MarshalCertificateToPEM(certs[0])
	if err != nil {
		return nil, fmt.Errorf("marshaling leaf cert: %w", err)
	}
	ret[fmt.Sprintf("%s_leaf.crt.pem", prefix)] = leaf

	// handle root
	root, err := cryptoutils.MarshalCertificateToPEM(certs[len(certs)-1])
	if err != nil {
		return nil, fmt.Errorf("marshaling root cert: %w", err)
	}
	ret[fmt.Sprintf("%s_root.crt.pem", prefix)] = root

	// handle intermediates
	for i, c := range certs[1 : len(certs)-1] {
		intermediate, err := cryptoutils.MarshalCertificateToPEM(c)
		if err != nil {
			return nil, fmt.Errorf("marshaling intermediate cert %d: %w", i, err)
		}
		ret[fmt.Sprintf("%s_intermediate_%d.crt.pem", prefix, i)] = intermediate
	}
	return ret, nil
}
