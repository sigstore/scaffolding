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
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
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

	v1Common "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/theupdateframework/go-tuf"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	if err := r.Init(false); err != nil {
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

// CreateRepo calls CreateRepoWithOptions, while setting:
// * CreateRepoOptions.AddMetadataTargets: true
// * CreateRepoOptions.AddTrustedRoot: false
func CreateRepo(ctx context.Context, files map[string][]byte) (tuf.LocalStore, string, error) {
	return CreateRepoWithOptions(ctx, files, CreateRepoOptions{AddMetadataTargets: true, AddTrustedRoot: false})
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

func constructTrustedRoot(targets []TargetWithMetadata) (*TargetWithMetadata, error) {
	tr := v1.TrustedRoot{
		MediaType: "application/vnd.dev.sigstore.trustedroot+json;version=0.1",
	}

	var fulcioLeaf, fulcioRoot, tsaLeaf, tsaRoot []byte
	var fulcioIntermed, tsaIntermed [][]byte

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
			case strings.Contains(target.Name, "leaf"):
				fulcioLeaf = target.Bytes
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
			tlinstance, err := pubkeyToTLogInstance(target.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse rekor key: %w", err)
			}
			tr.Tlogs = []*v1.TransparencyLogInstance{tlinstance}
		case CTFETarget:
			tlinstance, err := pubkeyToTLogInstance(target.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse ctlog key: %w", err)
			}
			tr.Ctlogs = []*v1.TransparencyLogInstance{tlinstance}
		}
	}
	var fulcioAuthority, tsaAuthority *v1.CertificateAuthority
	var err error

	// concat the fulcio chain and process it into CertificateAuthority
	fulcioAuthority, err = certChainToAuthority(concatCertChain(fulcioLeaf, fulcioIntermed, fulcioRoot))
	if err != nil {
		return nil, fmt.Errorf("failed to parse cert chain for Fulcio: %w", err)
	}
	tr.CertificateAuthorities = []*v1.CertificateAuthority{fulcioAuthority}

	// concat the tsa chain and process it into CertificateAuthority
	tsaAuthority, err = certChainToAuthority(concatCertChain(tsaLeaf, tsaIntermed, tsaRoot))
	if err != nil {
		return nil, fmt.Errorf("failed to parse cert chain for TSA: %w", err)
	}
	tr.TimestampAuthorities = append(tr.TimestampAuthorities, tsaAuthority)

	marshaller := &protojson.MarshalOptions{
		Indent: "  ",
	}
	serialized, err := marshaller.Marshal(&tr)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize trust root: %w", err)
	}

	return &TargetWithMetadata{
		Name:  "trusted_root.json",
		Bytes: serialized,
	}, nil
}

func concatCertChain(leaf []byte, intermediate [][]byte, root []byte) []byte {
	var result []byte
	result = append(result, leaf...)
	result = append(result, byte('\n'))
	for _, intermed := range intermediate {
		result = append(result, intermed...)
		result = append(result, byte('\n'))
	}
	result = append(result, root...)
	return result
}

func pubkeyToTLogInstance(key []byte) (*v1.TransparencyLogInstance, error) {
	logId := sha256.Sum256(key)
	der, _ := pem.Decode(key)
	keyDetails, err := getKeyDetails(der.Bytes)
	if err != nil {
		return nil, err
	}

	return &v1.TransparencyLogInstance{
		BaseUrl:       "",
		HashAlgorithm: v1Common.HashAlgorithm_SHA2_256, // TODO: make it possible to change this value
		PublicKey: &v1Common.PublicKey{
			RawBytes:   der.Bytes,
			KeyDetails: keyDetails,
			ValidFor: &v1Common.TimeRange{
				Start: timestamppb.New(time.Now()),
			},
		},
		LogId: &v1Common.LogId{
			KeyId: logId[:],
		},
	}, nil
}

func getKeyDetails(key []byte) (v1Common.PublicKeyDetails, error) {
	var k any
	var err1, err2 error

	k, err1 = x509.ParsePKCS1PublicKey(key)
	if err1 != nil {
		k, err2 = x509.ParsePKIXPublicKey(key)
		if err2 != nil {
			return 0, fmt.Errorf("Can't parse public key with PKCS1 or PKIX: %w, %w", err1, err2)
		}
	}

	// borrowed from https://github.com/kommendorkapten/trtool/blob/main/cmd/trtool/app/common.go
	switch v := k.(type) {
	case *ecdsa.PublicKey:
		if v.Curve == elliptic.P256() {
			return v1Common.PublicKeyDetails_PKIX_ECDSA_P256_SHA_256, nil
		}
		if v.Curve == elliptic.P384() {
			return v1Common.PublicKeyDetails_PKIX_ECDSA_P384_SHA_384, nil
		}
		if v.Curve == elliptic.P521() {
			return v1Common.PublicKeyDetails_PKIX_ECDSA_P521_SHA_512, nil
		}
		return 0, errors.New("unsupported elliptic curve")
	case *rsa.PublicKey:
		/*
			NOTE: It is not possible to recognize padding from just the public key alone;
			we will just assume that the padding used is pkcs1v15
			if padding == RSAPSS {
				switch v.Size() * 8 {
				case 2048:
					return v1Common.PublicKeyDetails_PKIX_RSA_PSS_2048_SHA256, nil
				case 3072:
					return v1Common.PublicKeyDetails_PKIX_RSA_PSS_3072_SHA256, nil
				case 4096:
					return v1Common.PublicKeyDetails_PKIX_RSA_PSS_4096_SHA256, nil
				default:
					return 0, fmt.Errorf("unsupported public modulus %d", v.Size())
				}
			}
		*/
		switch v.Size() * 8 {
		case 2048:
			return v1Common.PublicKeyDetails_PKIX_RSA_PKCS1V15_2048_SHA256, nil
		case 3072:
			return v1Common.PublicKeyDetails_PKIX_RSA_PKCS1V15_3072_SHA256, nil
		case 4096:
			return v1Common.PublicKeyDetails_PKIX_RSA_PKCS1V15_4096_SHA256, nil
		default:
			return 0, fmt.Errorf("unsupported public modulus %d", v.Size())
		}
	case ed25519.PublicKey:
		return v1Common.PublicKeyDetails_PKIX_ED25519, nil
	default:
		return 0, errors.New("unknown public key type")
	}
}

func certChainToAuthority(certChainPem []byte) (*v1.CertificateAuthority, error) {
	var cert *x509.Certificate
	var err error
	rest := certChainPem
	certChain := v1Common.X509CertificateChain{Certificates: []*v1Common.X509Certificate{}}

	// skip potential whitespace at end of file (8 is kinda random, but seems to work fine)
	for len(rest) > 8 {
		var derCert *pem.Block
		derCert, rest = pem.Decode(rest)
		cert, err = x509.ParseCertificate(derCert.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Fulcio certificate: %w", err)
		}
		certChain.Certificates = append(certChain.Certificates, &v1Common.X509Certificate{RawBytes: cert.Raw})
	}

	// we end up using information from the last certificate, which is the root
	uri := ""
	if len(cert.URIs) > 0 {
		uri = cert.URIs[0].String()
	}
	subject := v1Common.DistinguishedName{}
	if len(cert.Subject.Organization) > 0 {
		subject.Organization = cert.Subject.Organization[0]
		subject.CommonName = cert.Subject.CommonName
	}

	authority := v1.CertificateAuthority{
		Subject: &subject,
		Uri:     uri,
		ValidFor: &v1Common.TimeRange{
			Start: timestamppb.New(cert.NotBefore),
			End:   timestamppb.New(cert.NotAfter),
		},
		CertChain: &certChain,
	}

	return &authority, nil
}

func getTargetUsage(name string) string {
	for _, knownTargetType := range []string{FulcioTarget, RekorTarget, CTFETarget, TSATarget} {
		if strings.Contains(name, strings.ToLower(knownTargetType)) {
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
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
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
