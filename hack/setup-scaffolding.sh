<<<<<<< HEAD
#!/usr/bin/env bash
# Copyright 2022 The Sigstore Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# Because setting up tuf requires peeking into several namespaces, we can't
# do that simply from a Job because a pod can't read secrets in the other
# namespaces that we need.
# Alternative would be to create a full blown controller and reconcile the
# secret from there, which might be something that we might consider doing
# anyways.
# But for now, trying to make forward progress here, we have to run this
# script _after_ creating the scaffolding and once it's up.

# So, after [this](https://github.com/sigstore/scaffolding/blob/main/getting-started.md#then-wait-for-the-jobs-that-setup-dependencies-to-finish) step completes, run this script.

# First create the job which will combine a tuf-system/tuf-secrets secret
# that holds all the information that the tuf server will need to create
# the tuf store.
ko apply -f ./config/tuf/createsecret

# Then wait for it to do it's job
kubectl wait --timeout=5m -n tuf-system --for=condition=Complete jobs createsecret

# Copy the necessary secrets to the tuf-system namespace
# TODO(vaikas): Just copy the bits we care about
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n fulcio-system get secrets fulcio-secret -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -

# Then launch the tuf server
ko apply -BRf ./config/tuf/server
||||||| parent of c641b2f (Create a secret that's used by the tuf server.)
=======
#!/usr/bin/env bash
# Copyright 2022 The Sigstore Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

# Because setting up tuf requires peeking into several namespaces, we can't
# do that simply from a Job because a pod can't read secrets in the other
# namespaces that we need.
# Alternative would be to create a full blown controller and reconcile the
# secret from there, which might be something that we might consider doing
# anyways.
# But for now, trying to make forward progress here, we have to run this
# script _after_ creating the scaffolding and once it's up.

# So, after [this](https://github.com/sigstore/scaffolding/blob/main/getting-started.md#then-wait-for-the-jobs-that-setup-dependencies-to-finish) step completes, run this script.

# First create the job which will combine a tuf-system/tuf-secrets secret
# that holds all the information that the tuf server will need to create
# the tuf store.
ko apply -f ./config/tuf/createsecret

# Then wait for it to do it's job
kubectl wait --timeout=5m -n tuf-system --for=condition=Complete jobs createsecret

# Copy the necessary secrets to the tuf-system namespace
# TODO(vaikas): Just copy the bits we care about
kubectl -n ctlog-system get secrets ctlog-public-key -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -
kubectl -n fulcio-system get secrets fulcio-secret -oyaml | sed 's/namespace: .*/namespace: tuf-system/' | kubectl apply -f -

# Then launch the tuf server
ko apply -BRf ./config/tuf/server
<<<<<<< HEAD

# Above however is currently failing with the following error, and I'm not
# entirely sure how to fix, so I'm just creating this as a checkpoint.
```
vaikas@villes-mbp scaffolding % kubectl -n tuf-system logs tuf-00003-deployment-57f574c94c-kgbhr tuf
{"level":"info","ts":1659224120.0725565,"logger":"fallback","caller":"createrepo/main.go:57","msg":"running create_repo Version: devel GitCommit: 46ebe1479b33e5e0ba1efd0a6d967a68d1bbab5a BuildDate: 2022-07-27T23:51:17"}
{"level":"info","ts":1659224120.1282668,"logger":"fallback","caller":"createrepo/main.go:148","msg":"Creating the FS in \"/tmp\""}
{"level":"info","ts":1659224120.1304667,"logger":"fallback","caller":"createrepo/main.go:152","msg":"Creating new repo in \"/tmp\""}
{"level":"error","ts":1659224120.3686304,"logger":"fallback","caller":"createrepo/main.go:191","msg":"Failed to SnashotWithExpires tuf: missing metadata file targets.json","stacktrace":"main.createRepo\n\tgithub.com/sigstore/scaffolding/cmd/tuf/createrepo/main.go:191\nmain.main\n\tgithub.com/sigstore/scaffolding/cmd/tuf/createrepo/main.go:85\nruntime.main\n\truntime/proc.go:250"}
{"level":"panic","ts":1659224120.369749,"logger":"fallback","caller":"createrepo/main.go:87","msg":"Creating repot: tuf: missing metadata file targets.json","stacktrace":"main.main\n\tgithub.com/sigstore/scaffolding/cmd/tuf/createrepo/main.go:87\nruntime.main\n\truntime/proc.go:250"}
panic: Creating repot: tuf: missing metadata file targets.json
```
>>>>>>> c641b2f (Create a secret that's used by the tuf server.)
||||||| parent of d5030ca (oops, forgot to update scaffolding script.)

# Above however is currently failing with the following error, and I'm not
# entirely sure how to fix, so I'm just creating this as a checkpoint.
```
vaikas@villes-mbp scaffolding % kubectl -n tuf-system logs tuf-00003-deployment-57f574c94c-kgbhr tuf
{"level":"info","ts":1659224120.0725565,"logger":"fallback","caller":"createrepo/main.go:57","msg":"running create_repo Version: devel GitCommit: 46ebe1479b33e5e0ba1efd0a6d967a68d1bbab5a BuildDate: 2022-07-27T23:51:17"}
{"level":"info","ts":1659224120.1282668,"logger":"fallback","caller":"createrepo/main.go:148","msg":"Creating the FS in \"/tmp\""}
{"level":"info","ts":1659224120.1304667,"logger":"fallback","caller":"createrepo/main.go:152","msg":"Creating new repo in \"/tmp\""}
{"level":"error","ts":1659224120.3686304,"logger":"fallback","caller":"createrepo/main.go:191","msg":"Failed to SnashotWithExpires tuf: missing metadata file targets.json","stacktrace":"main.createRepo\n\tgithub.com/sigstore/scaffolding/cmd/tuf/createrepo/main.go:191\nmain.main\n\tgithub.com/sigstore/scaffolding/cmd/tuf/createrepo/main.go:85\nruntime.main\n\truntime/proc.go:250"}
{"level":"panic","ts":1659224120.369749,"logger":"fallback","caller":"createrepo/main.go:87","msg":"Creating repot: tuf: missing metadata file targets.json","stacktrace":"main.main\n\tgithub.com/sigstore/scaffolding/cmd/tuf/createrepo/main.go:87\nruntime.main\n\truntime/proc.go:250"}
panic: Creating repot: tuf: missing metadata file targets.json
```
=======
>>>>>>> d5030ca (oops, forgot to update scaffolding script.)
