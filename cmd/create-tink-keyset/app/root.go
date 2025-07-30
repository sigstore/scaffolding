// Copyright 2025 The Sigstore Authors.
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

package app

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"log/slog"
	"os"
	"strings"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tink-crypto/tink-go-gcpkms/v2/integration/gcpkms"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/proto/tink_go_proto"
	"github.com/tink-crypto/tink-go/v2/signature"
	tinkecdsa "github.com/tink-crypto/tink-go/v2/signature/ecdsa"
	tinked25519 "github.com/tink-crypto/tink-go/v2/signature/ed25519"
)

/*
Example command:

go run ./cmd/create-tink-keyset \
  --key-template ED25519 \
  --out enc-keyset.cfg \
  --key-encryption-key-uri gcp-kms://projects/project/locations/us-west1/keyRings/keyring/cryptoKeys/keyname \
  --public-key-out key.pem
*/

var rootCmd = &cobra.Command{
	Use:   "create-tink-keyset",
	Short: "Create a Tink keyset",
	Long:  "Generate a Tink keyset to be used to sign checkpoints, encrypted with a provided KMS key. Only supported for GCP currently.",
	Run: func(_ *cobra.Command, _ []string) {
		if viper.GetString("key-template") == "" {
			slog.Error("must provide --key-template for signing key algorithm")
			os.Exit(1)
		}
		kekURI := viper.GetString("key-encryption-key-uri")
		if kekURI == "" {
			slog.Error("must provide --key-encryption-key-uri for the GCP KMS CryptoKey resource that encrypts the keyset")
			os.Exit(1)
		}
		if !strings.HasPrefix(kekURI, "gcp-kms://") {
			slog.Error("--key-encryption-key-uri only supports GCP and the URI must begin with gcp-kms://")
			os.Exit(1)
		}
		if viper.GetString("out") == "" {
			slog.Error("must provide --out for output path of keyset")
			os.Exit(1)
		}
		if viper.GetString("public-key-out") == "" {
			slog.Error("must provide --public-key-out for output path of public key")
			os.Exit(1)
		}

		ctx := context.Background()

		// Generate GCP KMS client
		kmsClient, err := gcpkms.NewClientWithOptions(ctx, kekURI)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		kekAEAD, err := kmsClient.GetAEAD(kekURI)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		// Create keyset handle, which initializes the signing key based on the provided template
		keyTemplate, ok := algToKeyTemplate[viper.GetString("key-template")]
		if !ok {
			slog.Error("unsupported key template provided")
			os.Exit(1)
		}
		newHandle, err := keyset.NewHandle(keyTemplate)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		// Encrypt signing key and generate keyset
		buf := new(bytes.Buffer)
		writer := keyset.NewJSONWriter(buf)
		if err := newHandle.Write(writer, kekAEAD); err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		f, err := os.Create(viper.GetString("out"))
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		defer f.Close()
		_, err = f.Write(buf.Bytes())
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		// Generate PEM-encoded public key
		publicHandle, err := newHandle.Public()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		keyEntry, err := publicHandle.Primary()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		var pemPubKey []byte
		switch publicKey := keyEntry.Key().(type) {
		case *tinked25519.PublicKey:
			pemPubKey, err = cryptoutils.MarshalPublicKeyToPEM(ed25519.PublicKey(publicKey.KeyBytes()))
			if err != nil {
				slog.Error(err.Error())
				os.Exit(1)
			}
		case *tinkecdsa.PublicKey:
			curve := algToCurve[viper.GetString("key-template")]
			pubKey, err := curve.NewPublicKey(publicKey.PublicPoint())
			if err != nil {
				slog.Error(err.Error())
				os.Exit(1)
			}
			pemPubKey, err = cryptoutils.MarshalPublicKeyToPEM(pubKey)
			if err != nil {
				slog.Error(err.Error())
				os.Exit(1)
			}
		}
		pubF, err := os.Create(viper.GetString("public-key-out"))
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		defer pubF.Close()
		_, err = pubF.Write(pemPubKey)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		slog.Info("generated Tink keyset")
	},
}

var algToKeyTemplate = map[string]*tink_go_proto.KeyTemplate{
	"ED25519":           signature.ED25519KeyTemplate(),
	"ECDSA_P256":        signature.ECDSAP256KeyTemplate(),
	"ECDSA_P384_SHA384": signature.ECDSAP384SHA384KeyTemplate(),
	"ECDSA_P521":        signature.ECDSAP521KeyTemplate(),
}

var algToCurve = map[string]ecdh.Curve{
	"ECDSA_P256":        ecdh.P256(),
	"ECDSA_P384_SHA384": ecdh.P384(),
	"ECDSA_P521":        ecdh.P521(),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().String("key-template", "", "tink key template for the signing algorithm. Valid values are ED25519, ECDSA_P256, ECDSA_P384_SHA384, and ECDSA_P521")
	rootCmd.Flags().String("key-encryption-key-uri", "", "Resource URI for the KMS key that encrypts the keyset. Only GCP is supported, and the URI must match gcp-kms://projects/*/locations/*/keyRings/*/cryptoKeys/*")
	rootCmd.Flags().String("out", "", "output path for the encrypted keyset")
	rootCmd.Flags().String("public-key-out", "", "output path for the PEM-encoded public key")

	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
