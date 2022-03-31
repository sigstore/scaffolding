# Setup Sigstore Scaffolding

This action installs sigstore components and wires them together for test
purposes on an existing cluster. Also verifies they are up and running by
doing a cosign sign/verify.
Will also set up the environment variables so that calling
cosign can be used for keyless signing / verification against the cluster.

## Usage

```yaml
- uses: sigstore/scaffolding/actions/setup@main
  with:
    # Scaffolding version. v0.2.3 by default.
    scaffolding-version: v0.2.3
```

## Scenarios

```yaml
steps:
- uses: sigstore/scaffolding/actions/setup@main
```

## Exported environmental variables.

The following environmental variables are exported.

 * REKOR_URL
   Where Rekor runs. For cosign commands, you should use this for --rekor-url
   flag
 * FULCIO_URL
   Where Fulcio runs. For cosign commands, you should use this for --fulcio-url
   flag
 * CTLOG_URL
   CTLog where Fulcio writes.
 * SIGSTORE_CT_LOG_PUBLIC_KEY_FILE
   Public key file location. Used by cosign to validate SCT coming back from
   Fulcio.
 * SIGSTORE_ROOT_FILE
   Alternate sigstore root file, since we are using non-standard root for
   sigstore components.
 * SIGSTORE_TRUST_REKOR_API_PUBLIC_KEY
   Necessary to be set to true so that we can validate against non-standard
   Rekor instance that we use above.
 * ISSUER_URL
   This is the URL for fetching OIDC tokens off the cluster that you can then use as inputs to --identity-token to cosign
 * OIDC_TOKEN
   This is an already fetched OIDC token. Convenience if your tests do
   not run longer than 10 minutes, which is how long this token is
   valid for. If your tests run longer than this is valid, you can use
   `ISSUER_URL` above to fetch a new token.
