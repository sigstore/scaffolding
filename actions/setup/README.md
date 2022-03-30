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
