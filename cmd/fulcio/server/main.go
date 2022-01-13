/*
Copyright 2021 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"

	"github.com/sigstore/fulcio/cmd/app"
)

func main() {
	app.Execute(context.Background())
}
