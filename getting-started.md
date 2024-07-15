# Getting Started

This document should allow you to stand up a fully functioning sigstore stack,
including:

 * Fulcio
 * Rekor
 * CTLog
 * Trillian - backing Rekor and CTLog
 * Tuf mirror

# Using scaffolding on your own GitHub actions

There's a reusable [action](./actions/setup/README.md) that you can use as is.

# Prerequisites

You need to install `yq`. You can do this like so:

```shell
go install github.com/mikefarah/yq/v4@latest
```

You also need [ko](https://ko.build/) a tool for building lighter, more secure container images.

```shell
go install github.com/google/ko@latest
```

There are further install options on the [ko website](https://ko.build/).

# Running locally on KinD

You should be able to install KinD and Knative bits by running (from head, after
cloning the repo):

```shell
./hack/setup-kind.sh
```

Or by downloading a release version of the script

```shell
curl -fLo /tmp/setup-kind.sh https://github.com/sigstore/scaffolding/releases/download/v0.7.1/setup-kind.sh
chmod u+x /tmp/setup-kind.sh
/tmp/setup-kind.sh
```

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
b1e3f3238f7a   registry:2                        "/entrypoint.sh /etc…"   15 minutes ago   Up 15 minutes               0.0.0.0:5000->5000/tcp, :::5001->5001/tcp   registry.local
```

So that's the running version of the registry, so first kill and then remove it:

```shell
docker rm -f b1e3f3238f7a
```

# Install sigstore-scaffolding pieces

## From the release

```shell
curl -Lo /tmp/setup-scaffolding-from-release.sh https://github.com/sigstore/scaffolding/releases/download/v0.7.1/setup-scaffolding-from-release.sh
chmod u+x /tmp/setup-scaffolding-from-release.sh
/tmp/setup-scaffolding-from-release.sh
```

## From checked out repo

If you're deploying to kind cluster created above, tell `ko` where it is, or
change to where you're deploying your images.

```shell
export KO_DOCKER_REPO=registry.local:5001/sigstore
```

```shell
./hack/setup-scaffolding.sh
```

# Outputs

The step above creates 5 namespaces:

 * trillian-system
 * ctlog-system
 * fulcio-system
 * rekor-system
 * tuf-system

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

 * `fulcio.fulcio-system.svc` for HTTP
or
 * `fulcio-grpc.fulcio-system.svc` for GRPC

## rekor-system namespace

`rekor-system` namespace contains [Rekor](https://github.com/sigstore/rekor)
and Rekor can be accessed in the cluster with:

 * `rekor.rekor-system.svc`

## tuf-system namespace

`tuf-system` namespace contains [TUF](https://theupdateframework.io/) root
mirror that we need to point cosign and other tools to, because the root
of trust is not the public cosign instance. Tuf can be accessed in the cluster
with:

 * `tuf.tuf-system.svc`

## default namespace

 To make it easier to test keyless signing without going through the browser
 based auth, you can install `OIDC issuer` on your _TEST_ cluster. It does no
 authentication, so do not install this on anything except your local kind test
 cluster. Just by doing a curl against it will give you an OIDC token that you
 can use as --identity-token on the calls with `cosign`.

 ```shell
 ko apply -BRf ./testdata/config/gettoken
 ```

## Accessing your new cluster endpoints

In order to access the services running in the cluster, we utilize
port-forwarding provided by Kubernetes.

### Network access

Setup port forwarding:

```shell
kubectl -n kourier-system port-forward service/kourier-internal 8080:80 &
```

### Adding localhost entries to make tools usable

Add the following entries to your `/etc/hosts` file

```txt
127.0.0.1 rekor.rekor-system.svc
127.0.0.1 fulcio.fulcio-system.svc
127.0.0.1 ctlog.ctlog-system.svc
127.0.0.1 gettoken.default.svc
127.0.0.1 tuf.tuf-system.svc
```

### Setting up environmental variables
Instead of having to specify these in various flags when calling cosign and long
URLs, let's create some up front:

```shell
export REKOR_URL=http://rekor.rekor-system.svc:8080
export FULCIO_URL=http://fulcio.fulcio-system.svc:8080
export FULCIO_GRPC_URL=http://fulcio-grpc.fulcio-system.svc:8080
export ISSUER_URL=http://gettoken.default.svc:8080
export TUF_MIRROR=http://tuf.tuf-system.svc:8080
```

### Setting up an OIDC issuer running on the cluster.

For testing keyless signing we need an OIDC token provider, so let's create one
that runs on the cluster and issues OIDC tokens.

```shell
ko apply -BRf ./testdata/config/gettoken
```

## Testing Your new Sigstore Kind Cluster

Let's first run a quick smoke test that does a cosign sign followed by making
sure that the rekor entry is created for it.

1) Get TUF root from the tuf-system namespace

```shell
kubectl -n tuf-system get secrets tuf-root -ojsonpath='{.data.root}' | base64 -d > ./root.json
```

2) Initialize cosign with our root.

```shell
cosign initialize --mirror $TUF_MIRROR --root ./root.json
```

An example invocation of this on my machine looked like this:

```shell
vaikas@villes-mbp scaffolding % cosign initialize --mirror $TUF_MIRROR --root ./root.json
Root status:
 {
	"local": "/Users/vaikas/.sigstore/root",
	"remote": "http://tuf.tuf-system.svc:8080",
	"metadata": {
		"root.json": {
			"version": 1,
			"len": 2178,
			"expiration": "04 Feb 23 23:28 UTC",
			"error": ""
		},
		"snapshot.json": {
			"version": 1,
			"len": 618,
			"expiration": "04 Feb 23 23:28 UTC",
			"error": ""
		},
		"targets.json": {
			"version": 1,
			"len": 1028,
			"expiration": "04 Feb 23 23:28 UTC",
			"error": ""
		},
		"timestamp.json": {
			"version": 1,
			"len": 619,
			"expiration": "04 Feb 23 23:28 UTC",
			"error": ""
		}
	},
	"targets": [
		"rekor.pub",
		"ctfe.pub",
		"fulcio_v1.crt.pem"
	]
}
```

If you have an image that you want to play with, great, you can also create
one easily like this (that gets then uploaded to our local registry):

```shell
KO_DOCKER_REPO=registry.local:5001/sigstore
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

```shell
cosign sign --rekor-url $REKOR_URL --fulcio-url $FULCIO_URL --yes --allow-insecure-registry $demoimage --identity-token $(curl -s $ISSUER_URL)
```

An example invocation from my local instance is like so:

```shell
vaikas@villes-mbp scaffolding % cosign sign --rekor-url $REKOR_URL --fulcio-url $FULCIO_URL --yes --allow-insecure-registry $demoimage --identity-token $(curl -s $ISSUER_URL)
Generating ephemeral keys...
Retrieving signed certificate...

        Note that there may be personally identifiable information associated with this signed artifact.
        This may include the email address associated with the account with which you authenticate.
        This information will be used for signing this artifact and will be stored in public transparency logs and cannot be removed later.
Successfully verified SCT...
tlog entry created with index: 0
Pushing signature to: registry.local:5001/sigstore/demo
```

Then let's verify the signature.

```shell
cosign verify --rekor-url $REKOR_URL --allow-insecure-registry $demoimage --certificate-identity=https://kubernetes.io/namespaces/default/serviceaccounts/default --certificate-oidc-issuer=https://kubernetes.default.svc.cluster.local
```

An example invocation from my local instance is like so:

```shell
vaikas@villes-mbp scaffolding % cosign verify --rekor-url $REKOR_URL --allow-insecure-registry $demoimage --certificate-identity=https://kubernetes.io/namespaces/default/serviceaccounts/default --certificate-oidc-issuer=https://kubernetes.default.svc.cluster.local
**Warning** Missing fallback target fulcio.crt.pem, skipping

Verification for registry.local:5001/sigstore/demo@sha256:b6cfc6e87706304be13f607b238d905db1096619c0217c82f4151117e0112025 --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - Any certificates were verified against the Fulcio roots.

[{"critical":{"identity":{"docker-reference":"registry.local:5000/sigstore/demo"},"image":{"docker-manifest-digest":"sha256:b6cfc6e87706304be13f607b238d905db1096619c0217c82f4151117e0112025"},"type":"cosign container image signature"},"optional":{"Bundle":{"SignedEntryTimestamp":"MEUCIQD8MdBVswffTOuubuvTHIWw4BMkOmUmgrQEavmAnWZ1MAIgSNO+gf4ldCql0botNgtb23RWPD4iYv0Qq93sheWf5wo=","Payload":{"body":"eyJhcGlWZXJzaW9uIjoiMC4w<SNIPPED_HERE_FOR_READABILITY>b45b4573e2a7e5f876bdff025b06f3243"}},"Issuer":"https://kubernetes.default.svc","Subject":"https://kubernetes.io/namespaces/default/serviceaccounts/default"}}]
```

And the `**Warning**` is just letting us know that there's no custom metadata
on TUF, and we fallback on the hard-coded names, and that's one of the ones we
expect for Fulcio (and the other is the one we use: fulcio_v1.crt.pem)

```shell
echo -n 'foobar test attestation' > ./predicate-file
cosign attest --predicate ./predicate-file --fulcio-url $FULCIO_URL --rekor-url $REKOR_URL --allow-insecure-registry --yes $demoimage --identity-token $(curl -s $ISSUER_URL)
```

An example invocation from my local instance:

```shell
vaikas@villes-mbp scaffolding % echo -n 'foobar test attestation' > ./predicate-file
cosign attest --predicate ./predicate-file --fulcio-url $FULCIO_URL --rekor-url $REKOR_URL --allow-insecure-registry --yes $demoimage --identity-token $(curl -s $ISSUER_URL)

Generating ephemeral keys...
Retrieving signed certificate...

        Note that there may be personally identifiable information associated with this signed artifact.
        This may include the email address associated with the account with which you authenticate.
        This information will be used for signing this artifact and will be stored in public transparency logs and cannot be removed later.
Successfully verified SCT...
Using payload from: ./predicate-file
tlog entry created with index: 1
```

And then finally let's verify the attestation we just created:

```shell
cosign verify-attestation --rekor-url $REKOR_URL --allow-insecure-registry $demoimage --certificate-identity=https://kubernetes.io/namespaces/default/serviceaccounts/default --certificate-oidc-issuer=https://kubernetes.default.svc.cluster.local
```

An example invocation from my local instance:

```shell
vaikas@villes-mbp scaffolding % cosign verify-attestation --rekor-url $REKOR_URL --allow-insecure-registry $demoimage --certificate-identity=https://kubernetes.io/namespaces/default/serviceaccounts/default --certificate-oidc-issuer=https://kubernetes.default.svc.cluster.local
**Warning** Missing fallback target fulcio.crt.pem, skipping

Verification for registry.local:5001/sigstore/demo@sha256:b6cfc6e87706304be13f607b238d905db1096619c0217c82f4151117e0112025 --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - Any certificates were verified against the Fulcio roots.
Certificate subject:  https://kubernetes.io/namespaces/default/serviceaccounts/default
Certificate issuer URL:  https://kubernetes.default.svc
{"payloadType":"application/vnd.in-toto+json","payload":"eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjAuMSIsInByZWRpY2F0ZVR5cGUiOiJjb3NpZ24uc2lnc3RvcmUuZGV2L2F0dGVzdGF0aW9uL3YxIiwic3ViamVjdCI6W3sibmFtZSI6InJlZ2lzdHJ5LmxvY2FsOjUwMDAvc2lnc3RvcmUvZGVtbyIsImRpZ2VzdCI6eyJzaGEyNTYiOiJiNmNmYzZlODc3MDYzMDRiZTEzZjYwN2IyMzhkOTA1ZGIxMDk2NjE5YzAyMTdjODJmNDE1MTExN2UwMTEyMDI1In19XSwicHJlZGljYXRlIjp7IkRhdGEiOiJmb29iYXIgdGVzdCBhdHRlc3RhdGlvbiIsIlRpbWVzdGFtcCI6IjIwMjItMDgtMDdUMDM6NTU6NDhaIn19","signatures":[{"keyid":"","sig":"MEUCIHXTVuffNLmCtnYg2AqCZ1YZfN87Ct3jL6Opx6ZA1czAAiEAs4BG3wEHP49Kg2YB+7gcFqg64J77aS/IDKb6sSbmRzU="}]}
```

And you can inspect the `payload` of the attestation by base64 decoding the payload, so for me:

```shell
vaikas@villes-mbp scaffolding % echo 'eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjAuMSIsInByZWRpY2F0ZVR5cGUiOiJjb3NpZ24uc2lnc3RvcmUuZGV2L2F0dGVzdGF0aW9uL3YxIiwic3ViamVjdCI6W3sibmFtZSI6InJlZ2lzdHJ5LmxvY2FsOjUwMDAvc2lnc3RvcmUvZGVtbyIsImRpZ2VzdCI6eyJzaGEyNTYiOiJiNmNmYzZlODc3MDYzMDRiZTEzZjYwN2IyMzhkOTA1ZGIxMDk2NjE5YzAyMTdjODJmNDE1MTExN2UwMTEyMDI1In19XSwicHJlZGljYXRlIjp7IkRhdGEiOiJmb29iYXIgdGVzdCBhdHRlc3RhdGlvbiIsIlRpbWVzdGFtcCI6IjIwMjItMDgtMDdUMDM6NTU6NDhaIn19' | base64 -d
{"_type":"https://in-toto.io/Statement/v0.1","predicateType":"cosign.sigstore.dev/attestation/v1","subject":[{"name":"registry.local:5001/sigstore/demo","digest":{"sha256":"b6cfc6e87706304be13f607b238d905db1096619c0217c82f4151117e0112025"}}],"predicate":{"Data":"foobar test attestation","Timestamp":"2022-08-07T03:55:48Z"}}%
```

Notice our predicate is `foobar test attestation` as was in our predicate file.

## Generating trusted_root.json

The TUF mirror in this stack does not serve a
[`trusted_root.json`](https://github.com/sigstore/protobuf-specs/blob/main/protos/sigstore_trustroot.proto)
target, but you can generate one to use with certain sigstore clients.

1. Download and install [trtool](https://github.com/kommendorkapten/trtool).

2. Use `cosign initialize` as described above to download targets from the TUF
   mirror.

3. Initialize the trusted root with the Fulcio CA:

```
./trtool init -ca ~/.sigstore/root/targets/fulcio_v1.crt.pem -ca-uri $FULCIO_URL -ca-start $(date -Iseconds) | jq > tr.1.json
```

4. Add the transparency log and certificate transparency log keys:

```
./trtool add -f tr.1.json -type ctlog -uri $CTLOG_URL -pem ~/.sigstore/root/targets/ctfe.pub -start $(date -Iseconds) | jq > tr.2.json
./trtool add -f tr.2.json -type tlog -uri $REKOR_URL -pem ~/.sigstore/root/targets/rekor.pub -start $(date -Iseconds) | jq > trusted_root.json
```

5. Now the trusted_root.json can be used as input for sigstore clients:

```
sigstore-go -trustedrootJSONpath trusted_root.json -tufTrustedRoot root.json -artifact=blob -expectedSAN=https://kubernetes.io/namespaces/default/serviceaccounts/default -expectedIssuer=https://kubernetes.default.svc.cluster.local bundle.json
```
