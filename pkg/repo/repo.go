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

package repo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/theupdateframework/go-tuf"
	"knative.dev/pkg/logging"
)

const (
	FulcioTarget  = "Fulcio"
	RekorTarget   = "Rekor"
	CTFETarget    = "CTFE"
	TSATarget     = "TSA"
	UnknownTarget = "Unknown"
)

type CreateRepoOptions struct {
	AddMetadataTargets bool
	AddTrustedRoot     bool
}

// TargetWithMetadata describes a TUF target with the given Name, Bytes, and
// CustomMetadata
type TargetWithMetadata struct {
	Name           string
	Bytes          []byte
	CustomMetadata []byte
}

type CustomMetadata struct {
	Usage  string `json:"usage"`
	Status string `json:"status"`
	URI    string `json:"uri"`
}

type sigstoreCustomMetadata struct {
	Sigstore CustomMetadata `json:"sigstore"`
}

// CreateRepoWithMetadata will create a TUF repo for Sigstore by adding targets
// to the Root with custom metadata.
func CreateRepoWithMetadata(ctx context.Context, targets []TargetWithMetadata) (tuf.LocalStore, string, error) {
	// TODO: Make this an in-memory fileystem.
	tmpDir := os.TempDir()
	dir := path.Join(tmpDir, "tuf")
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create tmp TUF dir: %w", err)
	}
	dir += "/"
	logging.FromContext(ctx).Infof("Creating the FS in %q", dir)
	local := tuf.FileSystemStore(dir, nil)

	// Create and commit a new TUF repo with the targets to the store.
	logging.FromContext(ctx).Infof("Creating new repo in %q", dir)
	r, err := tuf.NewRepoIndent(local, "", " ")
	if err != nil {
		return nil, "", fmt.Errorf("failed to NewRepoIndent: %w", err)
	}

	if err := r.Init(true); err != nil {
		return nil, "", fmt.Errorf("failed to Init repo: %w", err)
	}

	// Make all metadata files expire in 6 months.
	expires := time.Now().AddDate(0, 6, 0)

	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		_, err := r.GenKeyWithExpires(role, expires)
		if err != nil {
			return nil, "", fmt.Errorf("failed to GenKeyWithExpires: %w", err)
		}
	}

	for _, t := range targets {
		logging.FromContext(ctx).Infof("Adding file: %s", t.Name)
		if err := writeStagedTarget(dir, t.Name, t.Bytes); err != nil {
			return nil, "", fmt.Errorf("failed to write staged target %s: %w", t.Name, err)
		}
		err = r.AddTargetWithExpires(t.Name, t.CustomMetadata, expires)
		if err != nil {
			return nil, "", fmt.Errorf("failed to add AddTargetWithExpires: %w", err)
		}
	}

	// Snapshot, Timestamp, and Publish the repository.
	if err := r.SnapshotWithExpires(expires); err != nil {
		return nil, "", fmt.Errorf("failed to add SnapShotWithExpires: %w", err)
	}
	if err := r.TimestampWithExpires(expires); err != nil {
		return nil, "", fmt.Errorf("failed to add TimestampWithExpires: %w", err)
	}
	if err := r.Commit(); err != nil {
		return nil, "", fmt.Errorf("failed to Commit: %w", err)
	}
	return local, dir, nil
}

// CreateRepoWithOptions creates and initializes a TUF repo for Sigstore by adding
// keys to bytes. keys are typically for a basic setup like:
// "fulcio_v1.crt.pem" - Fulcio root cert in PEM format
// "ctfe.pub" - CTLog public key in PEM format
// "rekor.pub" - Rekor public key in PEM format
// "tsa_leaf.crt.pem" - TSA leaf certificate in PEM format
// "tsa_intermediate_0.crt.pem" - TSA Intermediate certificate in PEM format
// "tsa_root.crt.pem" - TSA Intermediate certificate in PEM format
// but additional keys can be added here.
//
// This will also deduce the Usage for the keys based off the filename:
// if the filename contains:
//   - `fulcio` = it will get Usage set to `Fulcio`
//   - `ctfe` = it will get Usage set to `CTFE`
//   - `rekor` = it will get Usage set to `Rekor`
//   - `tsa` = it will get Usage set to `tsa`.
//   - Anything else will get set to `Unknown`
//
// The targets will be added individually to the TUF repo if CreateRepoOptions.AddMetadataTargets
// is set to true. The trusted_root.json file will be added if CreateRepoOptions.AddTrustedRoot
// is set to true. At least one of these has to be true.
func CreateRepoWithOptions(ctx context.Context, files map[string][]byte, options CreateRepoOptions) (tuf.LocalStore, string, error) {
	if !options.AddMetadataTargets && !options.AddTrustedRoot {
		return nil, "", errors.New("failed to create TUF repo: At least one of metadataTargets, trustedRoot must be true")
	}

	metadataTargets := make([]TargetWithMetadata, 0, len(files))
	for name, bytes := range files {
		scmActive, err := json.Marshal(&sigstoreCustomMetadata{Sigstore: CustomMetadata{Usage: getTargetUsage(name), Status: "Active"}})
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal custom metadata for %s: %w", name, err)
		}
		metadataTargets = append(metadataTargets, TargetWithMetadata{
			Name:           name,
			Bytes:          bytes,
			CustomMetadata: scmActive,
		})
	}

	targets := make([]TargetWithMetadata, 0, len(files)+1)
	if options.AddMetadataTargets {
		targets = append(targets, metadataTargets...)
	}
	if options.AddTrustedRoot {
		trustedRootTarget, err := constructTrustedRoot(metadataTargets)
		if err != nil {
			return nil, "", fmt.Errorf("failed to construct trust root: %w", err)
		}
		targets = append(targets, *trustedRootTarget)
	}

	return CreateRepoWithMetadata(ctx, targets)
}

// CreateRepo calls CreateRepoWithOptions, while setting:
// * CreateRepoOptions.AddMetadataTargets: true
// * CreateRepoOptions.AddTrustedRoot: false
func CreateRepo(ctx context.Context, files map[string][]byte) (tuf.LocalStore, string, error) {
	return CreateRepoWithOptions(ctx, files, CreateRepoOptions{AddMetadataTargets: true, AddTrustedRoot: true})
}

func constructTrustedRoot(targets []TargetWithMetadata) (*TargetWithMetadata, error) {
	var fulcioRoot, tsaLeaf, tsaRoot []byte
	var fulcioIntermed, tsaIntermed [][]byte
	rekorKeys := map[string]*root.TransparencyLog{}
	ctlogKeys := map[string]*root.TransparencyLog{}
	now := time.Now()

	// we sort the targets by Name, this results in intermediary certs being sorted correctly,
	// as long as there is less than 10, which is ok to assume for the purposes of this code
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Name < targets[j].Name
	})

	for _, target := range targets {
		// NOTE: in the below switch, we are able to process whole certificate chains, but we also support
		// if they're passed in as individual certificates, already split in individual targets
		switch getTargetUsage(target.Name) {
		case FulcioTarget:
			switch {
			// no leaf for Fulcio certificate, the leaf is the code signing cert
			case strings.Contains(target.Name, "intermediate"):
				fulcioIntermed = append(fulcioIntermed, target.Bytes)
			default:
				fulcioRoot = target.Bytes
			}
		case TSATarget:
			switch {
			case strings.Contains(target.Name, "leaf"):
				tsaLeaf = target.Bytes
			case strings.Contains(target.Name, "intermediate"):
				tsaIntermed = append(tsaIntermed, target.Bytes)
			default:
				tsaRoot = target.Bytes
			}
		case RekorTarget:
			tlinstance, id, err := pubkeyToTransparencyLogInstance(target.Bytes, now)
			if err != nil {
				return nil, fmt.Errorf("failed to parse rekor key: %w", err)
			}
			rekorKeys[id] = tlinstance
		case CTFETarget:
			tlinstance, id, err := pubkeyToTransparencyLogInstance(target.Bytes, now)
			if err != nil {
				return nil, fmt.Errorf("failed to parse ctlog key: %w", err)
			}
			ctlogKeys[id] = tlinstance
		}
	}

	fulcioChainPem := concatCertChain([]byte{}, fulcioIntermed, fulcioRoot)
	fulcioAuthorities := []root.CertificateAuthority{}
	if len(fulcioChainPem) > 0 {
		fulcioAuthority, err := certChainToCertificateAuthority(fulcioChainPem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cert chain for Fulcio: %w", err)
		}
		fulcioAuthorities = append(fulcioAuthorities, *fulcioAuthority)
	}

	tsaChainPem := concatCertChain(tsaLeaf, tsaIntermed, tsaRoot)
	tsaAuthorities := []root.CertificateAuthority{}
	if len(tsaChainPem) > 0 {
		tsaAuthority, err := certChainToCertificateAuthority(tsaChainPem)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cert chain for TSA: %w", err)
		}
		tsaAuthorities = append(tsaAuthorities, *tsaAuthority)
	}

	tr, err := root.NewTrustedRoot(
		root.TrustedRootMediaType01,
		fulcioAuthorities,
		ctlogKeys,
		tsaAuthorities,
		rekorKeys,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create TrustedRoot: %w", err)
	}
	serialized, err := json.Marshal(tr)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize TrustedRoot to JSON: %w", err)
	}

	return &TargetWithMetadata{
		Name:  "trusted_root.json",
		Bytes: serialized,
	}, nil
}

func pubkeyToTransparencyLogInstance(keyBytes []byte, tm time.Time) (*root.TransparencyLog, string, error) {
	logID := sha256.Sum256(keyBytes)
	der, _ := pem.Decode(keyBytes)
	key, keyDetails, err := getKeyWithDetails(der.Bytes)
	if err != nil {
		return nil, "", err
	}

	return &root.TransparencyLog{
		BaseURL:             "",
		ID:                  logID[:],
		ValidityPeriodStart: tm,
		HashFunc:            crypto.SHA256, // we can't get this from the keyBytes, assume SHA256
		PublicKey:           key,
		SignatureHashFunc:   keyDetails,
	}, hex.EncodeToString(logID[:]), nil
}

func getKeyWithDetails(key []byte) (crypto.PublicKey, crypto.Hash, error) {
	var k any
	var hashFunc crypto.Hash
	var err1, err2 error

	k, err1 = x509.ParsePKCS1PublicKey(key)
	if err1 != nil {
		k, err2 = x509.ParsePKIXPublicKey(key)
		if err2 != nil {
			return 0, 0, fmt.Errorf("can't parse public key with PKCS1 or PKIX: %w, %w", err1, err2)
		}
	}

	switch v := k.(type) {
	case *ecdsa.PublicKey:
		switch v.Curve {
		case elliptic.P256():
			hashFunc = crypto.SHA256
		case elliptic.P384():
			hashFunc = crypto.SHA384
		case elliptic.P521():
			hashFunc = crypto.SHA512
		default:
			return 0, 0, fmt.Errorf("unsupported elliptic curve %T", v.Curve)
		}
	case *rsa.PublicKey:
		switch v.Size() * 8 {
		case 2048, 3072, 4096:
			hashFunc = crypto.SHA256
		default:
			return 0, 0, fmt.Errorf("unsupported public modulus %d", v.Size())
		}
	case ed25519.PublicKey:
		hashFunc = crypto.SHA512
	default:
		return 0, 0, errors.New("unknown public key type")
	}

	return k, hashFunc, nil
}

func certChainToCertificateAuthority(certChainPem []byte) (*root.CertificateAuthority, error) {
	var cert *x509.Certificate
	var err error
	rest := bytes.TrimSpace(certChainPem)
	certChain := []*x509.Certificate{}

	for len(rest) > 0 {
		var derCert *pem.Block
		derCert, rest = pem.Decode(rest)
		rest = bytes.TrimSpace(rest)
		if derCert == nil {
			return nil, fmt.Errorf("input is left, but it is not a certificate: %+v", rest)
		}
		cert, err = x509.ParseCertificate(derCert.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		certChain = append(certChain, cert)
	}
	if len(certChain) == 0 {
		return nil, fmt.Errorf("no certificates found in input")
	}

	ca := root.CertificateAuthority{}

	for i, cert := range certChain {
		switch {
		case i == 0 && !cert.IsCA:
			ca.Leaf = cert
		case i < len(certChain)-1:
			ca.Intermediates = append(ca.Intermediates, cert)
		case i == len(certChain)-1:
			ca.Root = cert
		}
	}

	ca.ValidityPeriodStart = certChain[0].NotBefore
	ca.ValidityPeriodEnd = certChain[0].NotAfter

	return &ca, nil
}

func concatCertChain(leaf []byte, intermediate [][]byte, root []byte) []byte {
	result := []byte{}
	if len(leaf) > 0 {
		// for Fulcio, the leaf will always be empty, don't necessarily append an empty newline
		result = append(result, leaf...)
		result = append(result, byte('\n'))
	}
	for _, intermed := range intermediate {
		result = append(result, intermed...)
		result = append(result, byte('\n'))
	}
	result = append(result, root...)
	return result
}

func getTargetUsage(name string) string {
	for _, knownTargetType := range []string{FulcioTarget, RekorTarget, CTFETarget, TSATarget} {
		if strings.Contains(strings.ToLower(name), strings.ToLower(knownTargetType)) {
			return knownTargetType
		}
	}

	return UnknownTarget
}

func writeStagedTarget(dir, path string, data []byte) error {
	path = filepath.Join(dir, "staged", "targets", path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	/* #nosec G306 */
	return os.WriteFile(path, data, 0644)
}

// CompressFS archives a TUF repository so that it can be written to Secret
// for later use.
func CompressFS(fsys fs.FS, buf io.Writer, skipDirs map[string]bool) error {
	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	err := fs.WalkDir(fsys, "repository", func(file string, d fs.DirEntry, err error) error {
		// Skip the 'keys' and 'staged' directory
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}

		// Stat the file to get the details of it.
		fi, err2 := fs.Stat(fsys, file)
		if err2 != nil {
			return fmt.Errorf("fs.Stat %s: %w", file, err2)
		}
		header, err2 := tar.FileInfoHeader(fi, file)
		if err2 != nil {
			return fmt.Errorf("FileInfoHeader %s: %w", file, err2)
		}
		header.Name = filepath.ToSlash(file)
		if err2 := tw.WriteHeader(header); err2 != nil {
			return err
		}
		// For files, write the contents.
		if !d.IsDir() {
			data, err2 := fsys.Open(file)
			if err2 != nil {
				return fmt.Errorf("opening %s: %w", file, err2)
			}
			if _, err2 := io.Copy(tw, data); err2 != nil {
				return fmt.Errorf("copying %s: %w", file, err2)
			}
		}
		return nil
	})

	if err != nil {
		tw.Close()
		zr.Close()
		return fmt.Errorf("WalkDir: %w", err)
	}

	if err := tw.Close(); err != nil {
		zr.Close()
		return fmt.Errorf("tar.NewWriter Close(): %w", err)
	}
	return zr.Close()
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

// Uncompress takes a TUF repository that's been compressed with Compress and
// writes to dst directory.
func Uncompress(src io.Reader, dst string) error {
	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := header.Name

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", target)
		}

		// add dst + re-format slashes according to system
		// #nosec G305
		// mitigated below
		target = filepath.Join(dst, header.Name)
		// this check is to mitigate gosec G305 (zip slip vulnerability)
		if !strings.HasPrefix(target, filepath.Clean(dst)) {
			return fmt.Errorf("%s: %s", "content filepath is tainted", header.Name)
		}
		// check the type
		switch header.Typeflag {
		// Create directories
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, os.ModePerm); err != nil {
					return err
				}
			}
		// Write out files
		case tar.TypeReg:
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			for {
				_, err := io.CopyN(fileToWrite, tr, 1024)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return err
				}
			}
			if err := fileToWrite.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", target, err)
			}
		}
	}
	return nil
}
