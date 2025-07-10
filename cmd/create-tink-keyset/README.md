# `create-tink-keyset`

This command generates a Tink keyset that can be used for signing. The generated keyset is encrypted with a Key Encryption Key (KEK) from a cloud KMS. Currently, only Google Cloud KMS is supported.

The command also outputs the corresponding public key in PEM format. This is useful for TUF, where the public key needs to be distributed for clients.

## Prerequisites

You must have a Key Encryption Key (KEK) available in Google Cloud KMS. You also need to have credentials configured locally to be able to access the KMS key.
You can initialize credentials with `gcloud auth application-default login`.

## Usage

```shell
go run ./cmd/create-tink-keyset [flags]
```

### Flags

*   `--key-template` **(required)**: The Tink key template for the signing algorithm.
    *   Valid values: `ED25519`, `ECDSA_P256`, `ECDSA_P384_SHA384`, `ECDSA_P521`.
*   `--key-encryption-key-uri` **(required)**: The resource URI for the KMS key that will encrypt the keyset.
    *   Only GCP is supported.
    *   The URI must be in the format `gcp-kms://projects/*/locations/*/keyRings/*/cryptoKeys/*`.
*   `--out` **(required)**: The output path for the encrypted Tink keyset file.
*   `--public-key-out` **(required)**: The output path for the PEM-encoded public key.

## Example

```shell
go run ./cmd/create-tink-keyset \
  --key-template ED25519 \
  --out enc-keyset.cfg \
  --key-encryption-key-uri gcp-kms://projects/my-gcp-project/locations/global/keyRings/my-keyring/cryptoKeys/my-kek \
  --public-key-out public.pem
```

## Outputs

The command generates two files:

1.  **`--out` file (e.g., `enc-keyset.cfg`)**: An encrypted Tink keyset in JSON format. This file contains the private key, encrypted by the specified GCP KMS key. This file should be kept private.
2.  **`--public-key-out` file (e.g., `public.pem`)**: The corresponding public key in PEM format, to verify signatures created with the private key from the keyset.
