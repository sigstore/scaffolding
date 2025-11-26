# `create-tink-keyset`

This command generates a Tink keyset that can be used for signing log checkpoints. The generated keyset is encrypted with a Key Encryption Key (KEK) from a cloud KMS. Currently, only Google Cloud KMS is supported.

The command also outputs the corresponding public key and the log ID, which are used when populating the TUF trusted root.

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
*   `--origin` **(required)**: The origin of the log. Used to generate the checkpoint key ID.
*   `--key-encryption-key-uri` **(required)**: The resource URI for the KMS key that will encrypt the keyset.
    *   Only GCP is supported.
    *   The URI must be in the format `gcp-kms://projects/*/locations/*/keyRings/*/cryptoKeys/*`.
*   `--out` **(required)**: The output path for the encrypted Tink keyset file.
*   `--public-key-out` **(required)**: The output path for the base64-encoded PKIX public key.
*   `--key-id-out` **(required)**: The output path for the base64-encoded checkpoint key ID.

## Example

```shell
go run ./cmd/create-tink-keyset \
  --key-template ED25519 \
  --origin my-log.sigstore.dev \
  --out enc-keyset.json \
  --key-encryption-key-uri gcp-kms://projects/my-gcp-project/locations/global/keyRings/my-keyring/cryptoKeys/my-kek \
  --public-key-out public.b64 \
  --key-id-out logid.b64
```

## Outputs

The command generates three files:

1.  **`--out` file (e.g., `enc-keyset.json`)**: An encrypted Tink keyset in JSON format. This file contains the private key, encrypted by the specified GCP KMS key. This file should be kept private.
2.  **`--public-key-out` file (e.g., `public.b64`)**: The corresponding public key, as a base64-encoded PKIX structure. This is used to verify signatures created with the private key from the keyset.
3.  **`--key-id-out` file (e.g., `logid.b64`)**: The base64-encoded log ID, which is a hash of the origin and the public key.
