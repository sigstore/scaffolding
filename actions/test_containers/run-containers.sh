#!/usr/bin/env bash
#
# Copyright 2025 The Sigstore Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

echo "setting up kind"
go install sigs.k8s.io/kind@v0.20.0
kind create cluster

docker_compose="docker compose"
if ! ${docker_compose} version >/dev/null 2>&1; then
    docker_compose="docker-compose"
fi

echo "setting up OIDC provider"
pushd ./fakeoidc
oidcimg=$(ko build main.go --local)
docker network ls | grep fulcio_default || docker network create fulcio_default --label "com.docker.compose.network=fulcio_default"
docker run -d --rm -p 8080:8080 --network fulcio_default --name fakeoidc $oidcimg
oidc_ip=$(docker inspect fakeoidc | jq -r '.[0].NetworkSettings.Networks.fulcio_default.IPAddress')
export OIDC_URL="http://${oidc_ip}:8080"
cat <<EOF > /tmp/fulcio-config.json
{
  "OIDCIssuers": {
    "$OIDC_URL": {
      "IssuerURL": "$OIDC_URL",
      "ClientID": "sigstore",
      "Type": "email"
    }
  }
}
EOF
popd

pushd $HOME

echo "downloading service repos"
for repo in rekor fulcio timestamp-authority; do
    if [[ ! -d $repo ]]; then
        git clone https://github.com/sigstore/${repo}.git
    else
        pushd $repo
        git pull
        popd
    fi
done

echo "starting services"
export FULCIO_METRICS_PORT=2113
export FULCIO_CONFIG=/tmp/fulcio-config.json
for repo in rekor fulcio timestamp-authority; do
    pushd $repo
    if [ "$repo" == "fulcio" ]; then
       yq -i e '.networks={"default":{ "name":"fulcio_default","external":true }}' docker-compose.yml
       yq -i e '.services.fulcio-server.networks=["default"]' docker-compose.yml
    fi
    ${docker_compose} up -d
    echo -n "waiting up to 60 sec for system to start"
    count=0
    if [ "$repo" == "timestamp-authority" ]; then
      target_healthy=1
    else
      target_healthy=3
    fi
    until [ $(${docker_compose} ps | grep -c "(healthy)") == $target_healthy ];
    do
        if [ $count -eq 18 ]; then
           echo "! timeout reached"
           exit 1
        else
           echo -n "."
           sleep 10
           let 'count+=1'
        fi
    done
    popd
done
