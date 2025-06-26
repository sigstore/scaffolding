// Copyright 2025 The Sigstore Authors
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

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/transparency-dev/tessera/api/layout"
)

// checkRekorV2CompleteFirstTile checks to see if the first tile in a shard is still a partial tile.
// If no, then we must fetch a partial tile. If yes, we must fetch a full tile.
// See https://github.com/C2SP/C2SP/issues/145.
func checkRekorV2CompleteFirstTile(rekorURL string) (bool, error) {
	req, err := retryablehttp.NewRequest("GET", rekorURL+"/api/v2/checkpoint", nil)
	if err != nil {
		return false, fmt.Errorf("invalid request for checkpoint: %w", err)
	}

	setHeaders(req, "", ReadProberCheck{})
	resp, err := retryableClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("unexpected error getting loginfo endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected response code received from loginfo endpoint: %w", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("reading loginfo body: %w", err)
	}

	// The second line will be the log size.
	// See https://github.com/C2SP/C2SP/blob/94c93bee35b922c91b3729c7f184ce3104a6c7cb/tlog-checkpoint.md#note-text.
	logSizeBytes := bytes.Split(bodyBytes, []byte{'\n'})[1]
	logSize, err := strconv.Atoi(string(logSizeBytes))
	if err != nil {
		return false, fmt.Errorf("parsing log size: %w", err)
	}
	return logSize > layout.TileWidth, nil
}

// determineRekorV2ShardCoverage determines which endpoints to check for a given rekorV2 shard host.
func determineRekorV2ShardCoverage(rekorURL string) ([]*ReadProberCheck, error) {
	hasCompleteFirtstTile, err := checkRekorV2CompleteFirstTile(rekorURL)
	if err != nil {
		return nil, err
	}
	proberChecks := []*ReadProberCheck{
		{
			Endpoint: "/api/v2/checkpoint",
			Method:   GET,
		},
	}
	if hasCompleteFirtstTile {
		proberChecks = append(proberChecks,
			&ReadProberCheck{
				Endpoint: "/api/v2/tile/entries/000",
				Method:   GET,
			},
			&ReadProberCheck{
				Endpoint: "/api/v2/tile/0/000",
				Method:   GET,
			},
		)
	} else {
		proberChecks = append(proberChecks,
			&ReadProberCheck{
				Endpoint: "/api/v2/tile/entries/000.p1/1",
				Method:   GET,
			},
			&ReadProberCheck{
				Endpoint: "/api/v2/tile/0/000.p1/1",
				Method:   GET,
			},
		)
	}
	return proberChecks, nil
}
