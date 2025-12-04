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

// This module contains helpers for manipulating CTLog configuration. Since the
// configuration is in a protobuf and we have to marshal/unmarshal, and update
// it during the operation of the CTLog / Fulcio, hoisted into here for easier
// testing.

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"knative.dev/pkg/logging"
)

const (
	// PrivateKey is the key in the map holding the encrypted PEM private key
	// for CTLog.
	PrivateKey = "private"
	// PublicKey is the key in the map holding the PEM public key for CTLog.
	PublicKey = "public"
	// FulcioKey is the key in the map holding the list of Fulcio certificates
	// for CTLog.
	FulcioKey = "fulcio"
	bitSize   = 4096
)

// Config abstracts the proto munging to/from bytes suitable for working
// with secrets / configmaps. Note that we keep fulcioCerts here though
// technically they are not part of the config, however because we create a
// secret/CM that we then mount, they need to be synced.
type Config struct {
	PrivKey crypto.PrivateKey
	PubKey  crypto.PublicKey

	// FulcioCerts contains one or more Root certificates for Fulcio.
	// It may contain more than one if Fulcio key is rotated for example, so
	// there will be a period of time when we allow both. It might also contain
	// multiple Root Certificates, if we choose to support admitting certificates from fulcio instances run by others
	FulcioCerts [][]byte
}

func extractFulcioRoot(fulcioRoot []byte) ([]byte, error) {
	// Fetch only root certificate from the chain
	certs, err := cryptoutils.UnmarshalCertificatesFromPEM(fulcioRoot)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal certficate chain: %w", err)
	}
	return cryptoutils.MarshalCertificateToPEM(certs[len(certs)-1])
}

// AddFulcioRoot will add the specified fulcioRoot to the list of trusted
// Fulcios. If it already exists, it's a nop.
// The fulcioRoot should come from the call to fetch a PublicFulcio root
// and is the ChainPEM from the fulcioclient RootResponse.
func (c *Config) AddFulcioRoot(ctx context.Context, fulcioRoot []byte) error {
	root, err := extractFulcioRoot(fulcioRoot)
	if err != nil {
		return fmt.Errorf("extracting fulcioRoot: %w", err)
	}
	for _, fc := range c.FulcioCerts {
		if bytes.Equal(fc, root) {
			logging.FromContext(ctx).Infof("Found existing fulcio root, not adding: %s", string(root))
			return nil
		}
	}
	logging.FromContext(ctx).Infof("Adding new FulcioRoot: %s", string(root))
	c.FulcioCerts = append(c.FulcioCerts, root)
	return nil
}

// RemoveFulcioRoot will remove the specified fulcioRoot from the list of
// trusted Fulcios. If
func (c *Config) RemoveFulcioRoot(ctx context.Context, fulcioRoot []byte) error {
	root, err := extractFulcioRoot(fulcioRoot)
	if err != nil {
		return fmt.Errorf("extracting fulcioRoot: %w", err)
	}

	newCerts := make([][]byte, 0, len(c.FulcioCerts))
	for _, fc := range c.FulcioCerts {
		if !bytes.Equal(fc, root) {
			newCerts = append(newCerts, fc)
		} else {
			logging.FromContext(ctx).Infof("Found existing fulcio root, removing: %s", string(root))
		}
	}
	c.FulcioCerts = newCerts
	return nil
}

func (c *Config) String() string {
	var sb strings.Builder
	for _, fulcioCert := range c.FulcioCerts {
		sb.WriteString(fmt.Sprintf("fulciocert:\n%s\n", string(fulcioCert)))
	}
	// Note this goofy cast to crypto.Signer since the any interface has no
	// methods so cast here so that we get the Public method which all core
	// keys support.
	if signer, ok := c.PrivKey.(crypto.Signer); ok {
		if marshaledPub, err := x509.MarshalPKIXPublicKey(signer.Public()); err == nil {
			pubPEM := pem.EncodeToMemory(
				&pem.Block{
					Type:  "PUBLIC KEY",
					Bytes: marshaledPub,
				},
			)
			sb.WriteString(fmt.Sprintf("PublicKey:\n%s\n", pubPEM))
		}
	}
	return sb.String()
}

// Unmarshal converts serialized (from secret, or configmap) form of the proto
// and secrets and constructs a CTLogConfig.
// Note however that because we do not update public/private keys once set
// we do not roundtrip these into their original forms.
func Unmarshal(_ context.Context, in map[string][]byte) (*Config, error) {
	var private, public []byte
	var ok bool
	if private, ok = in[PrivateKey]; !ok {
		return nil, fmt.Errorf("missing entry for %s", PrivateKey)
	}
	if public, ok = in[PublicKey]; !ok {
		return nil, fmt.Errorf("missing entry for %s", PublicKey)
	}

	var err error
	ret := Config{}
	ret.PubKey, err = cryptoutils.UnmarshalPEMToPublicKey(public)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	ret.PrivKey, _, err = parsePrivateKey(private)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %w", err)
	}

	if roots, ok := in[FulcioKey]; ok {
		rest := roots
		for len(rest) != 0 {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				return nil, fmt.Errorf("invalid fulcio roots: %w", err)
			}
			ret.FulcioCerts = append(ret.FulcioCerts, pem.EncodeToMemory(block))
		}
	}
	return &ret, nil
}

// MarshalConfig marshals the CTLogConfig into a format that can be handed
// to the CTLog in form of a secret. Returns a map with the
// following keys:
// private - CTLog private key, PEM encoded and encrypted with the password
// public - CTLog public key, PEM encoded
// fulcio-%d - For each fulcioCerts, contains one entry so we can support
// multiple.
func (c *Config) MarshalConfig() (map[string][]byte, error) {
	secrets, err := c.marshalSecrets()
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// MarshalSecrets returns a map suitable for creating a secret out of
// containing the following keys:
// private - CTLog private key, PEM encoded
// public - CTLog public key, PEM encoded
// fulcio-%d - For each fulcioCerts, contains one entry so we can support
// multiple.
func (c *Config) marshalSecrets() (map[string][]byte, error) {
	var marshalledPrivKey []byte
	var err error
	var blockType string
	switch k := c.PrivKey.(type) {
	case *rsa.PrivateKey:
		blockType = "PRIVATE KEY"
		// Encode private key to PKCS #8 ASN.1 PEM.
		marshalledPrivKey, err = x509.MarshalPKCS8PrivateKey(k)
	case *ecdsa.PrivateKey:
		blockType = "EC PRIVATE KEY"
		marshalledPrivKey, err = x509.MarshalECPrivateKey(k)
	default:
		return nil, fmt.Errorf("unrecognized private key type %T", k)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	block := &pem.Block{
		Type:  blockType,
		Bytes: marshalledPrivKey,
	}

	privPEM := pem.EncodeToMemory(block)
	if privPEM == nil {
		return nil, fmt.Errorf("failed to encode encrypted private key")
	}
	// Encode public key to PKIX ASN.1 PEM.
	var pubkey crypto.Signer
	var ok bool

	// Note this goofy cast to crypto.Signer since the any interface has no
	// methods so cast here so that we get the Public method which all core
	// keys support.
	if pubkey, ok = c.PrivKey.(crypto.Signer); !ok {
		return nil, fmt.Errorf("failed to convert private key to crypto.Signer")
	}

	marshalledPubKey, err := x509.MarshalPKIXPublicKey(pubkey.Public())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: marshalledPubKey,
		},
	)
	data := map[string][]byte{
		PrivateKey: privPEM,
		PublicKey:  pubPEM,
	}
	for _, cert := range c.FulcioCerts {
		data[FulcioKey] = append(data[FulcioKey], cert...)
	}
	return data, nil
}

// ParseExistingPrivateKey reads in a private key bytes and returns private, public keys for it.
func ParseExistingPrivateKey(privateKey []byte) (crypto.PrivateKey, crypto.PublicKey, error) {
	priv, signer, err := parsePrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}

	return priv, signer.Public(), nil
}

func parsePrivateKey(privateKey []byte) (crypto.PrivateKey, crypto.Signer, error) {
	privPEM, _ := pem.Decode(privateKey)
	if privPEM == nil {
		return nil, nil, fmt.Errorf("did not find valid private PEM data")
	}

	var priv crypto.PrivateKey
	var err error
	if priv, err = x509.ParsePKCS8PrivateKey(privPEM.Bytes); err != nil {
		// Try it as RSA
		if priv, err = x509.ParsePKCS1PrivateKey(privPEM.Bytes); err != nil {
			if priv, err = x509.ParseECPrivateKey(privPEM.Bytes); err != nil {
				return nil, nil, fmt.Errorf("failed to parse private key PEM: %w", err)
			}
		}
	}
	var ok bool
	var signer crypto.Signer
	if signer, ok = priv.(crypto.Signer); !ok {
		return nil, nil, errors.New("failed to convert private key to Signer")
	}
	return priv, signer, nil
}
