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
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/google/certificate-transparency-go/trillian/ctfe/configpb"
	"github.com/google/trillian/crypto/keyspb"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"knative.dev/pkg/logging"
)

const (
	// ConfigKey is the key in the map holding the marshalled CTLog config.
	ConfigKey = "config"
	// PrivateKey is the key in the map holding the encrypted PEM private key
	// for CTLog.
	PrivateKey = "private"
	// PublicKey is the key in the map holding the PEM public key for CTLog.
	PublicKey = "public"
	// LegacyRootCAKey is the key for when we only supported a single entry
	// in the config.
	LegacyRootCAKey = "rootca"
	bitSize         = 4096

	// This is hardcoded since this is where we mount the certs in the
	// container.
	rootsPemFileDir = "/ctfe-keys/"
	// This file contains the private key for the CTLog
	privateKeyFile = "/ctfe-keys/private"
)

// Config abstracts the proto munging to/from bytes suitable for working
// with secrets / configmaps. Note that we keep fulcioCerts here though
// technically they are not part of the config, however because we create a
// secret/CM that we then mount, they need to be synced.
type Config struct {
	PrivKey         crypto.PrivateKey
	PrivKeyPassword string
	PubKey          crypto.PublicKey
	LogID           int64
	LogPrefix       string

	// Address of the gRPC Trillian Admin Server (host:port)
	TrillianServerAddr string

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
	sb.WriteString(fmt.Sprintf("PrivateKeyPassword: %s\n", c.PrivKeyPassword))
	sb.WriteString(fmt.Sprintf("LogID: %d\n", c.LogID))
	sb.WriteString(fmt.Sprintf("LogPrefix: %s\n", c.LogPrefix))
	sb.WriteString(fmt.Sprintf("TrillianServerAddr: %s\n", c.TrillianServerAddr))
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
	var config, private, public []byte
	var ok bool
	if config, ok = in[ConfigKey]; !ok {
		return nil, fmt.Errorf("missing entry for %s", ConfigKey)
	}
	if private, ok = in[PrivateKey]; !ok {
		return nil, fmt.Errorf("missing entry for %s", PrivateKey)
	}
	if public, ok = in[PublicKey]; !ok {
		return nil, fmt.Errorf("missing entry for %s", PublicKey)
	}
	multiConfig := configpb.LogMultiConfig{}
	if err := prototext.Unmarshal(config, &multiConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	// We only have one backend specified for us, so even though multiconfig
	// can have many, we'll have only one.
	if multiConfig.LogConfigs == nil {
		return nil, fmt.Errorf("missing multiConfig or nil LogConfigs")
	}
	if len(multiConfig.LogConfigs.Config) != 1 {
		return nil, fmt.Errorf("unexpected number of LogConfig, want 1 got %d", len(multiConfig.LogConfigs.Config))
	}
	ret := Config{}
	logConfig := multiConfig.GetLogConfigs().Config[0]
	ret.LogID = logConfig.LogId
	ret.LogPrefix = logConfig.Prefix

	if multiConfig.Backends == nil {
		return nil, fmt.Errorf("missing backends")
	}
	if len(multiConfig.Backends.GetBackend()) != 1 {
		return nil, fmt.Errorf("unexpected number of Backends, want 1 got %d", len(multiConfig.Backends.Backend))
	}
	ret.TrillianServerAddr = multiConfig.Backends.GetBackend()[0].GetBackendSpec()

	// Then we need to decode public key
	var err error
	ret.PubKey, err = cryptoutils.UnmarshalPEMToPublicKey(public)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	privProto, err := logConfig.PrivateKey.UnmarshalNew()
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	pb, ok := privProto.(*keyspb.PEMKeyFile)
	if !ok {
		return nil, fmt.Errorf("not a valid PEMKeyFile in proto")
	}

	ret.PrivKeyPassword = pb.Password

	ret.PrivKey, _, err = DecryptExistingPrivateKey(private, ret.PrivKeyPassword)
	if err != nil {
		return nil, fmt.Errorf("decrypting existing private key: %w", err)
	}
	// Make sure to dedupe along the way just to make sure we do not have
	// duplicate entries.
	uniqueFulcioCerts := map[string][]byte{}

	// If there's legacy rootCA entry, check it first. This will get converted
	// to fulcio-0 when marshaling, but we just want to make sure it's there
	// when we're converting from ConfigMap based configuration into secret
	// based one.
	if legacyRoot, ok := in[LegacyRootCAKey]; ok && len(legacyRoot) > 0 {
		uniqueFulcioCerts[string(legacyRoot)] = legacyRoot
	}

	for k, v := range in {
		if strings.HasPrefix(k, "fulcio-") {
			uniqueFulcioCerts[string(v)] = v
		}
	}

	// Then loop through Fulcio roots that have been deduped above
	for _, v := range uniqueFulcioCerts {
		ret.FulcioCerts = append(ret.FulcioCerts, v)
	}
	return &ret, nil
}

// MarshalConfig marshals the CTLogConfig into a format that can be handed
// to the CTLog in form of a secret or configmap. Returns a map with the
// following keys:
// config - CTLog configuration
// private - CTLog private key, PEM encoded and encrypted with the password
// public - CTLog public key, PEM encoded
// fulcio-%d - For each fulcioCerts, contains one entry so we can support
// multiple.
func (c *Config) MarshalConfig(ctx context.Context) (map[string][]byte, error) {
	// Since we can have multiple Fulcio secrets, we need to construct a set
	// of files containing them for the RootsPemFile. Names don't matter
	// so we just call them fulcio-%
	// What matters however is to ensure that the filenames match the keys
	// in the configmap / secret that we construct so they get properly mounted.
	rootPems := make([]string, 0, len(c.FulcioCerts))
	for i := range c.FulcioCerts {
		rootPems = append(rootPems, fmt.Sprintf("%sfulcio-%d", rootsPemFileDir, i))
	}

	var pubkey crypto.Signer
	var ok bool
	// Note this goofy cast to crypto.Signer since the any interface has no
	// methods so cast here so that we get the Public method which all core
	// keys support.
	if pubkey, ok = c.PrivKey.(crypto.Signer); !ok {
		logging.FromContext(ctx).Fatalf("Failed to convert private key to crypto.Signer")
	}
	keyDER, err := x509.MarshalPKIXPublicKey(pubkey.Public())
	if err != nil {
		logging.FromContext(ctx).Panicf("Failed to marshal the public key: %v", err)
	}
	proto := configpb.LogConfig{
		LogId:        c.LogID,
		Prefix:       c.LogPrefix,
		RootsPemFile: rootPems,
		PrivateKey: mustMarshalAny(&keyspb.PEMKeyFile{
			Path:     privateKeyFile,
			Password: c.PrivKeyPassword}),
		PublicKey:      &keyspb.PublicKey{Der: keyDER},
		LogBackendName: "trillian",
		ExtKeyUsages:   []string{"CodeSigning"},
	}

	multiConfig := configpb.LogMultiConfig{
		LogConfigs: &configpb.LogConfigSet{
			Config: []*configpb.LogConfig{&proto},
		},
		Backends: &configpb.LogBackendSet{
			Backend: []*configpb.LogBackend{{
				Name:        "trillian",
				BackendSpec: c.TrillianServerAddr,
			}},
		},
	}
	marshalledConfig, err := prototext.Marshal(&multiConfig)
	if err != nil {
		return nil, err
	}
	secrets, err := c.marshalSecrets()
	if err != nil {
		return nil, err
	}
	secrets[ConfigKey] = marshalledConfig
	return secrets, nil
}

// MarshalSecrets returns a map suitable for creating a secret out of
// containing the following keys:
// private - CTLog private key, PEM encoded and encrypted with the password
// public - CTLog public key, PEM encoded
// fulcio-%d - For each fulcioCerts, contains one entry so we can support
// multiple.
func (c *Config) marshalSecrets() (map[string][]byte, error) {
	// Encode private key to PKCS #8 ASN.1 PEM.
	marshalledPrivKey, err := x509.MarshalPKCS8PrivateKey(c.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshalledPrivKey,
	}
	// Encrypt the pem
	encryptedBlock, err := x509.EncryptPEMBlock(rand.Reader, block.Type, block.Bytes, []byte(c.PrivKeyPassword), x509.PEMCipherAES256) // nolint
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(encryptedBlock)
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
	for i, cert := range c.FulcioCerts {
		fulcioKey := fmt.Sprintf("fulcio-%d", i)
		data[fulcioKey] = cert
	}
	return data, nil
}

func mustMarshalAny(pb proto.Message) *anypb.Any {
	ret, err := anypb.New(pb)
	if err != nil {
		panic(fmt.Sprintf("MarshalAny failed: %v", err))
	}
	return ret
}

// DecryptExistingPrivateKey reads in an encrypted private key, decrypts with
// the given password, and returns private, public keys for it.
func DecryptExistingPrivateKey(privateKey []byte, password string) (crypto.PrivateKey, crypto.PublicKey, error) {
	privPEM, _ := pem.Decode(privateKey)
	if privPEM == nil {
		return nil, nil, fmt.Errorf("did not find valid private PEM data")
	}
	privatePEMBlock, err := x509.DecryptPEMBlock(privPEM, []byte(password))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt private PEMKeyFile: %w", err)
	}

	var priv crypto.PrivateKey
	if priv, err = x509.ParsePKCS8PrivateKey(privatePEMBlock); err != nil {
		// Try it as RSA
		if priv, err = x509.ParsePKCS1PrivateKey(privatePEMBlock); err != nil {
			if priv, err = x509.ParseECPrivateKey(privatePEMBlock); err != nil {
				return nil, nil, fmt.Errorf("failed to parse private key PEM: %w", err)
			}
		}
	}
	var ok bool
	var signer crypto.Signer
	if signer, ok = priv.(crypto.Signer); !ok {
		return nil, nil, errors.New("failed to convert private key to Signer")
	}

	return priv, signer.Public(), nil
}
