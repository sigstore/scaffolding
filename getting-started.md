# Getting Started

This document should allow you to stand up a fully functioning sigstore stack,
including:

 * Fulcio
 * Rekor
 * CTLog
 * Trillian - backing Rekor and CTLog

# Using scaffolding on your own GitHub actions

There's a reusable [action](./actions/setup/README.md) that you can use as is.

# Prerequisites

You need to install `yq`. You can do this like so:
```
go install github.com/mikefarah/yq/v4@latest
```

# Running locally on KinD

You should be able to install KinD and Knative bits by running (from head, after
cloning the repo):

```shell
./hack/setup-kind.sh
```

Or by downloading a release version of the script
```shell
curl -Lo /tmp/setup-kind.sh https://github.com/sigstore/scaffolding/releases/download/v0.3.0/setup-kind.sh
chmod u+x /tmp/setup-kind.sh
/tmp/setup-kind.sh
```

**NOTE** For Macs the airplay receiver uses the 5000 port and may need to be
disabled, details [here](https://developer.apple.com/forums/thread/682332)).
Alternatively, you can manually modify the script and change the
[REGISTRY_PORT](https://github.com/sigstore/scaffolding/blob/main/hack/setup-kind.sh#L19)

*NOTE* If you run the script multiple times, you will have to delete the cluster
and uninstall the
docker registry container between running the setup-kind.sh it spins up a
registry container in a daemon mode.
To clean a previously running registry, you can do one of these:

YOLO:

```shell
docker rm -f `docker ps -a | grep 'registry:2' | awk -F " " '{print $1}'`
```

Or to check things first:

```shell
docker ps -a | grep registry
b1e3f3238f7a   registry:2                        "/entrypoint.sh /etc…"   15 minutes ago   Up 15 minutes               0.0.0.0:5000->5000/tcp, :::5000->5000/tcp   registry.local
```

So that's the running version of the registry, so first kill and then remove it:
```shell
docker rm -f b1e3f3238f7a
```

# Install sigstore-scaffolding pieces

```shell
curl -L https://github.com/sigstore/scaffolding/releases/download/v0.3.0/release.yaml | kubectl apply -f -
```

# Then wait for the jobs that setup dependencies to finish

```shell
kubectl wait --timeout=15m -A --for=condition=Complete jobs --all
```

Obviously if you have other jobs running, you might have to tune this, for deets
see [below](#outputs) what gets deployed and where. See below for how to
test / use the local instance.

 # Outputs

The deployment above creates 4 namespaces:

 * trillian-system
 * ctlog-system
 * fulcio-system
 * rekor-system

## trillian-system namespace

`trillian-system` namespace contains all things related to
[Trillian](https://github.com/google/trillian). This means there will be two
services `log-server`, `log-signer`, and a mysql pod.

To access these services from the cluster, you'd use:

 * `log-server.trillian-system.svc`
 * `log-signer.trillian-system.svc`
 * `mysql-trillian.trillian-system.svc`

 ## ctlog-system namespace

 `ctlog-system` namespace contains the
 [ctlog](https://github.com/google/certificate-transparency-go) service and
 can be accessed with:

  * `ctlog.ctlog-system.svc`

## fulcio-system namespace

`fulcio-system` namespace contains [Fulcio](https://github.com/sigstore/fulcio)
and Fulcio can be accessed in the cluster with:

 * `fulcio.fulcio-system.svc`

## rekor-system namespace

`rekor-system` namespace contains [Rekor](https://github.com/sigstore/rekor)
and Rekor can be accessed in the cluster with:

 * `rekor.rekor-system.svc`

 ## default namespace

 To make it easier to test keyless signing without going through the browser
 based auth, there's an `OIDC issuer` installed on the cluster. Just by doing
 a curl against it will give you an OIDC token that you can use as
 --identity-token on the calls with `cosign`

## Testing Your new Sigstore Kind Cluster

Let's first run a quick smoke test that does a cosign sign followed by making
sure that the rekor entry is created for it.

1) Get ctlog-public-key and add to default namespace
```shell
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: default/' | kubectl apply -f -
```

2) Get fulcio-secret and add to default namespace
```shell
kubectl -n fulcio-system get secrets fulcio-secret -oyaml | sed 's/namespace: .*/namespace: default/' | kubectl apply -f -
```

3) Create the three test jobs (checktree, sign-job, and verify-job)  using this
yaml (this may take a bit (~couple of minutes), since the jobs are launched
simultaneously)
```shell
curl -L https://github.com/sigstore/scaffolding/releases/download/v0.3.0/testrelease.yaml | kubectl apply -f -
```

3) To view if jobs have completed
```shell
kubectl wait --timeout=5m --for=condition=Complete jobs checktree sign-job verify-job
```

## Exercising the local cluster

Because all the pieces are running in the kind cluster, we need to make couple
of things to make it usable by normal cosign tooling from your local machine.

### Certificates

There are two certificates that we need, CT Log and Fulcio root certs. Note that
if you are switching back and forth between public / your instance, you might
not want to export these variables as hilarity will ensue.

CT Log:
```shell
kubectl -n ctlog-system get secrets ctlog-public-key -o=jsonpath='{.data.public}' | base64 -d > ./ctlog-public.pem
export SIGSTORE_CT_LOG_PUBLIC_KEY_FILE=./ctlog-public.pem
```

Fulcio root:
```shell
kubectl -n fulcio-system get secrets fulcio-secret -ojsonpath='{.data.cert}' | base64 -d > ./fulcio-root.pem
export SIGSTORE_ROOT_FILE=./fulcio-root.pem
```

### Network access

Setup port forwarding:

```shell
kubectl -n kourier-system port-forward service/kourier-internal 8080:80 &
```

### Adding localhost entries to make tools usable

Add the following entries to your `/etc/hosts` file

```
127.0.0.1 rekor.rekor-system.svc
127.0.0.1 fulcio.fulcio-system.svc
127.0.0.1 ctlog.ctlog-system.svc
127.0.0.1 gettoken.default.svc
```

This makes using tooling easier, for example:

```shell
rekor-cli --rekor_server http://rekor.rekor-system.svc:8080 loginfo
```

For example, this is what I get after smoke tests have successfully completed:
```shell
rekor-cli --rekor_server http://rekor.rekor-system.svc:8080 loginfo
No previous log state stored, unable to prove consistency
Verification Successful!
Tree Size: 1
Root Hash: 062e2fa50e2b523f9cfd4eadc4b67745436226d64bf9799d57c5dc023681c4b8
Timestamp: 2022-02-04T22:09:46Z
```

You can then execute various cosign/rekor-cli commands against these. However,
until [this issue](https://github.com/sigstore/cosign/issues/1405) gets fixed
for cosign you have to use `--allow-insecure-flag` in your cosign invocations.

Instead of having to specify various ENV flags, when calling cosign and long
URLs, let's create some up front:

```
export REKOR_URL=http://rekor.rekor-system.svc:8080
export FULCIO_URL=http://fulcio.fulcio-system.svc:8080
export ISSUER_URL=http://gettoken.default.svc:8080
# Since we run our own Rekor, when we are verifying things, we need to fetch
# the Rekor Public Key. This flag allows for that.
export SIGSTORE_TRUST_REKOR_API_PUBLIC_KEY=1
# This one is necessary to perform keyless signing with Fulcio.
export COSIGN_EXPERIMENTAL=1
```

If you have an image that you want to play with, great, you can also create
one easily like this (that gets then uploaded to our local registry):

```
KO_DOCKER_REPO=registry.local:5000/knative
pushd $(mktemp -d)
go mod init example.com/demo
cat <<EOF > main.go
package main
import "fmt"
func main() {
   fmt.Println("hello world")
}
EOF
demoimage=`ko publish -B example.com/demo`
export demoimage=$demoimage
echo Created image $demoimage
popd
```

Then let's sign it (or change $demoimage to something else).

```
cosign sign --rekor-url $REKOR_URL --fulcio-url $FULCIO_URL --force --allow-insecure-registry $demoimage --identity-token `curl -s $ISSUER_URL`
```

An example invocation from my local instance is like so:

```
vaikas@villes-mbp cosign % cosign sign --rekor-url $REKOR_URL --fulcio-url $FULCIO_URL --force --allow-insecure-registry $demoimage --identity-token `curl -s $ISSUER_URL`
Handling connection for 8080
Generating ephemeral keys...
Retrieving signed certificate...
Handling connection for 8080
**Warning** Using a non-standard public key for verifying SCT: ./ctlog-public.pem
Successfully verified SCT...
Handling connection for 8080
tlog entry created with index: 4
Pushing signature to: registry.local:5000/knative/demo
```

Then let's verify the signature.

```
./cosign verify --rekor-url $REKOR_URL --allow-insecure-registry $demoimage
```

An example invocation from my local instance is like so:

```
vaikas@villes-mbp cosign % cosign verify --rekor-url $REKOR_URL --allow-insecure-registry $demoimage
**Warning** Using a non-standard public key for Rekor: ./rekor-public.pem

Verification for registry.local:5000/knative/demo@sha256:6c6fd6a4115c6e998ff357cd914680931bb9a6c1a7cd5f5cb2f5e1c0932ab6ed --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - Any certificates were verified against the Fulcio roots.

[{"critical":{"identity":{"docker-reference":"registry.local:5000/knative/demo"},"image":{"docker-manifest-digest":"sha256:6c6fd6a4115c6e998ff357cd914680931bb9a6c1a7cd5f5cb2f5e1c0932ab6ed"},"type":"cosign container image signature"},"optional":{"Bundle":{"SignedEntryTimestamp":"MEYCIQC7nD8O7J79X2yx/Jj1Jd0YNOMZHvtfF8czrwVZs68TjgIhAJBvz5fIy/54f0ozScRZUu0h/aVxEp60shasI/mKmfgx","Payload":{"body":"eyJhcGlWZX<<SNIPPED HERE FOR READABILITYUzB0Q2c9PSJ9fX19","integratedTime":1649358969,"logIndex":4,"logID":"77f6de90a6672a37e47286c96c4a7ae0a18dc224403dd6dc7567604a99658c1c"}},"Issuer":"https://kubernetes.default.svc","Subject":"https://kubernetes.io/namespaces/default/serviceaccounts/default"}}]
```

And the `**Warning**` is just letting us know that we're using a different
SCT than the public instance, which we are :)

Then let's create an attestation for it:

```
echo -n 'foobar test attestation' > ./predicate-file
cosign attest --predicate ./predicate-file --fulcio-url $FULCIO_URL --rekor-url $REKOR_URL --allow-insecure-registry --force $demoimage --identity-token `curl -s $ISSUER_URL`
```

An example invocation from my local instance:

```
vaikas@villes-mbp cosign % cosign attest --predicate ./predicate-file --fulcio-url $FULCIO_URL --rekor-url $REKOR_URL --allow-insecure-registry --force $demoimage --identity-token `curl -s $ISSUER_URL`
Handling connection for 8080
Generating ephemeral keys...
Retrieving signed certificate...
Handling connection for 8080
**Warning** Using a non-standard public key for verifying SCT: ./ctlog-public.pem
Successfully verified SCT...
Using payload from: ./predicate-file
Handling connection for 8080
tlog entry created with index: 5
```

And then finally let's verify the attestation we just created:

```
cosign verify-attestation --rekor-url $REKOR_URL --allow-insecure-registry $demoimage
```

An example invocation from my local instance:

```
vaikas@villes-mbp cosign % cosign verify-attestation --rekor-url $REKOR_URL --allow-insecure-registry $demoimage
**Warning** Using a non-standard public key for Rekor: ./rekor-public.pem
Right before checking policies
Verification for registry.local:5000/knative/demo@sha256:6c6fd6a4115c6e998ff357cd914680931bb9a6c1a7cd5f5cb2f5e1c0932ab6ed --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - Any certificates were verified against the Fulcio roots.
Certificate subject:  https://kubernetes.io/namespaces/default/serviceaccounts/default
Certificate issuer URL:  https://kubernetes.default.svc
{"payloadType":"application/vnd.in-toto+json","payload":"eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjAuMSIsInByZWRpY2F0ZVR5cGUiOiJjb3NpZ24uc2lnc3RvcmUuZGV2L2F0dGVzdGF0aW9uL3YxIiwic3ViamVjdCI6W3sibmFtZSI6InJlZ2lzdHJ5LmxvY2FsOjUwMDAva25hdGl2ZS9kZW1vIiwiZGlnZXN0Ijp7InNoYTI1NiI6IjZjNmZkNmE0MTE1YzZlOTk4ZmYzNTdjZDkxNDY4MDkzMWJiOWE2YzFhN2NkNWY1Y2IyZjVlMWMwOTMyYWI2ZWQifX1dLCJwcmVkaWNhdGUiOnsiRGF0YSI6ImZvb2JhciB0ZXN0IGF0dGVzdGF0aW9uIiwiVGltZXN0YW1wIjoiMjAyMi0wNC0wN1QxOToyMjoyNVoifX0=","signatures":[{"keyid":"","sig":"MEUCIQC/slGQVpRKgw4Jo8tcbgo85WNG/FOJfxcvQFvTEnG9swIgP4LeOmID+biUNwLLeylBQpAEgeV6GVcEpyG6r8LVnfY="}]}
```

And you can inspect the payload of the attestation by base64 decoding the payload, so for me:

```
vaikas@villes-mbp cosign % echo 'eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjAuMSIsInByZWRpY2F0ZVR5cGUiOiJjb3NpZ24uc2lnc3RvcmUuZGV2L2F0dGVzdGF0aW9uL3YxIiwic3ViamVjdCI6W3sibmFtZSI6InJlZ2lzdHJ5LmxvY2FsOjUwMDAva25hdGl2ZS9kZW1vIiwiZGlnZXN0Ijp7InNoYTI1NiI6IjZjNmZkNmE0MTE1YzZlOTk4ZmYzNTdjZDkxNDY4MDkzMWJiOWE2YzFhN2NkNWY1Y2IyZjVlMWMwOTMyYWI2ZWQifX1dLCJwcmVkaWNhdGUiOnsiRGF0YSI6ImZvb2JhciB0ZXN0IGF0dGVzdGF0aW9uIiwiVGltZXN0YW1wIjoiMjAyMi0wNC0wN1QxOToyMjoyNVoifX0=' | base64 -d
{"_type":"https://in-toto.io/Statement/v0.1","predicateType":"cosign.sigstore.dev/attestation/v1","subject":[{"name":"registry.local:5000/knative/demo","digest":{"sha256":"6c6fd6a4115c6e998ff357cd914680931bb9a6c1a7cd5f5cb2f5e1c0932ab6ed"}}],"predicate":{"Data":"foobar test attestation","Timestamp":"2022-04-07T19:22:25Z"}}%
```

Notice our predicate is `foobar test attestation` as was in our predicate file.
