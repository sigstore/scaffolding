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
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
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
	rest := chain
	var next *pem.Block
	i := 0
	for {
		next, rest = pem.Decode(rest)
		if next == nil {
			// No PEM found, return error
			return nil, errors.New("no valid PEM found in chain")
		}
		if rest == nil || len(rest) == 0 {
			// This is the last one. We're going to call it root
			ret[fmt.Sprintf("%s_root.crt.pem", prefix)] = pem.EncodeToMemory(next)
			break
		} else {
			if i == 0 {
				// Leaf
				ret[fmt.Sprintf("%s_leaf.crt.pem", prefix)] = pem.EncodeToMemory(next)
			} else {
				ret[fmt.Sprintf("%s_intermediate_%d.crt.pem", prefix, i-1)] = pem.EncodeToMemory(next)
			}
			i++
		}
	}
	return ret, nil
}
