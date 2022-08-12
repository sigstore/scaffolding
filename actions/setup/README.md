# Setup Sigstore Scaffolding

This action installs sigstore components and wires them together for test
purposes on an existing cluster. Also verifies they are up and running by
doing a cosign sign/verify.

Will also set up URL endpoints for Rekor, Fulcio, and TUF so that you can use
for example `cosign` to test against this installation of Sigstore. You have
to initialize `cosign` explicitly like this:

```shell
cosign initialize --mirror $TUF_MIRROR --root ./root.json
```

**Deprecated**
For older versions of Scaffolding, you can set `legacy-variables` to "true",
which is currently the default. When setting this you will not use TUF but
instead, specify the various public keys and certs via local files and
use environment variables to point to them. The legacy way with environment
variables will be removed in the future, not only from Scaffolding but
likely from Cosign.

## Usage

```yaml
- uses: sigstore/scaffolding/actions/setup@main
  with:
    # Scaffolding version. 'latest-release' by default.
    scaffolding-version: "v0.4.3"
    # Do not set the deprecated environment variables, use TUF instead
    legacy-variables: "false"
```

## Scenarios

```yaml
steps:
- uses: sigstore/scaffolding/actions/setup@main
```

## Exported environmental variables.

The following environmental variables are exported.

 * TUF_MIRROR
   This is the TUF mirror installed on to the cluster. You have to then
   initialize cosign like so:
   ```shell
   cosign initialize --mirror $TUF_MIRROR --root ./root.json
   ```
 * REKOR_URL
   Where Rekor runs. For cosign commands, you should use this for --rekor-url
   flag
 * FULCIO_URL
   Where Fulcio runs. For cosign commands, you should use this for
   --fulcio-url flag
 * CTLOG_URL
   CTLog Fulcio uses for Certificate Transparency Log.
 * ISSUER_URL
   This is the URL for fetching OIDC tokens off the cluster that you can then use as inputs to --identity-token to `cosign`
 * OIDC_TOKEN
   This is an already fetched OIDC token. Convenience if your tests do
   not run longer than 10 minutes, which is how long this token is
   valid for. If your tests run longer than this is valid, you can use
   `ISSUER_URL` above to fetch a new token by specifying
   --identity-token=$(curl -s $ISSUER_URL)
 * SIGSTORE_CT_LOG_PUBLIC_KEY_FILE **DEPRECATED**
   Public key file location. Used by cosign to validate SCT coming back from
   Fulcio.
 * SIGSTORE_ROOT_FILE **DEPRECATED**
   Alternate sigstore root file, since we are using non-standard root for
   sigstore components.
 * SIGSTORE_REKOR_API_PUBLIC_KEY **DEPRECATED**
   Necessary to be set with the location of the public key file, so that we can validate against non-standard
   Rekor instance that we use above.

## TUF Root file
The action also creates a ./root.json that contains the TUF root that you use
to initialize cosign with.
