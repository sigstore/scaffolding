# Getting Started

This document should allow you to stand up a fully functioning sigstore stack,
including:

 * Fulcio
 * Rekor
 * CTLog
 * Trillian - backing Rekor and CTLog

# Using scaffolding on your own GitHub actions

If you want to just incorporate into your tests, you can see how Tekton chains
does it [here](https://github.com/tektoncd/chains/blob/main/.github/workflows/kind-e2e.yaml#L76-L104) or
looking at [E2E](./github/workflows/fulcio-rekor-kind.yaml) test that spins all
these up.

As part of the E2E test we use [cosign](https://github.com/sigstore/cosign) to
sign an image (and verify an entry made it Rekor), that should hopefully show
you to use it in your tests as well. The invocation is
[here](./testdata/config/sign-job/sign-job.yaml) and while it's wrapped in a k8s
Job and it uses a container, it basically executes this against the stack
deployed above:

```shell
COSIGN_EXPERIMENTAL=true SIGSTORE_CT_LOG_PUBLIC_KEY_FILE=/var/run/sigstore-root/rootfile.pem
cosign sign --fulcio-url=http://fulcio.fulcio-system.svc \
--rekor-url=http://rekor.rekor-system.svc \
ko://github.com/sigstore/scaffolding/cmd/rekor/checktree
```

Where the `rootfile.pem` gets mounted by the job, and it's the public key of the
CTLog, so we can verify the SCT coming back from Fulcio.


But roughly the workflow in your Github Action should be along these lines. It
setups a KinD cluster as well as sigstore and makes sure the setup works in
three distinct steps.

```
    - name: Setup Cluster
      run: |
        curl -Lo ./setup-kind.sh https://github.com/sigstore/scaffolding/releases/download/${{ env.SIGSTORE_SCAFFOLDING_RELEASE_VERSION }}/setup-kind.sh
        chmod u+x ./setup-kind.sh
        ./setup-kind.sh \
          --registry-url $(echo ${KO_DOCKER_REPO} | cut -d'/' -f 1) \
          --cluster-suffix cluster.local \
          --k8s-version ${{ matrix.k8s-version }} \
          --knative-version ${KNATIVE_VERSION}
    - name: Install sigstore pieces
      timeout-minutes: 10
      run: |
        curl -L https://github.com/sigstore/scaffolding/releases/download/${{ env.SIGSTORE_SCAFFOLDING_RELEASE_VERSION }}/release.yaml | kubectl apply -f -
        # Wait for all the ksvc to be up.
        kubectl wait --timeout 10m -A --for=condition=Ready ksvc --all
    - name: Run sigstore tests to make sure all is well
      run: |
        # Grab the secret from the ctlog-system namespace and make a copy
        # in our namespace so we can get access to the CT Log public key
        # so we can verify the SCT coming from there.
        kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: default/' | kubectl apply -f -
        curl -L https://github.com/sigstore/scaffolding/releases/download/${{ env.SIGSTORE_SCAFFOLDING_RELEASE_VERSION }}/testrelease.yaml | kubectl create -f -
        kubectl wait --for=condition=Complete --timeout=90s job/check-oidc
        kubectl wait --for=condition=Complete --timeout=90s job/checktree
```

Rest of this document talks about howto run locally on KinD.

# Running locally on KinD

You should be able to install KinD and Knative bits by running (from head, after
cloning the repo):

```shell
./hack/setup-kind.sh
```

Or by downloading a release version of the script
```shell
curl -Lo /tmp/setup-kind.sh https://github.com/sigstore/scaffolding/releases/download/v0.2.0/setup-kind.sh
chmod u+x /tmp/setup-kind.sh
/tmp/setup-kind.sh
```

**NOTE** For Macs the airplay receiver uses the 5000 port and may need to be
disabled, details [here](https://developer.apple.com/forums/thread/682332)).
Alternatively, you can manually modify the script and change the
[REGISTRY_PORT](https://github.com/sigstore/scaffolding/blob/main/hack/setup-mac-kind.sh#L19)

*NOTE* If you run the script multiple times, you will have to uninstall the
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
curl -L https://github.com/sigstore/scaffolding/releases/download/v0.2.0/release.yaml | kubectl apply -f -
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

## Testing Your new Sigstore Kind Cluster

Let's first run a quick smoke test that does a cosign sign followed by making
sure that the rekor entry is created for it.

1) Get ctlog-public-key and add to default namespace
```shell
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: default/' | kubectl apply -f -
```

2) Create the two test jobs (checktree and check-oidc)  using this yaml (this may take a bit (~couple of minutees), since the two jobs are launched simultaneously)
```shell
curl -L https://github.com/sigstore/scaffolding/releases/download/v0.1.19/testrelease.yaml | kubectl apply -f -
```

3) To view if jobs have completed
```shell
kubectl wait --timeout=5m --for=condition=Complete jobs checktree check-oidc
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
For example, to verify an image hosted in the local registry:

```shell
COSIGN_EXPERIMENTAL=1 ./main verify --allow-insecure-registry  registry.local:5000/knative/pythontest@sha256:080c3ad99fdd8b6f23da3085fb321d8a4fa57f8d4dd30135132e0fe3b31aa602
```
