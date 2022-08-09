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

THIS_OS="$(uname -s)"
echo "RUNNING ON ${THIS_OS}"
if [ "${THIS_OS}" == "Darwin" ]; then
  echo "Running on Darwin"
  RUNNING_ON_MAC="true"
else
  RUNNING_ON_MAC="false"
fi

# Check required utilities are on path
for i in "yq" "kubectl"
do
  command -v $i >/dev/null 2>&1 || { echo >&2 "$i not found"; exit 1; }
done

# Defaults
K8S_VERSION="v1.21.x"
KNATIVE_VERSION="1.1.0"
REGISTRY_NAME="registry.local"
REGISTRY_PORT="5000"
CLUSTER_SUFFIX="cluster.local"

while [[ $# -ne 0 ]]; do
  parameter="$1"
  case "${parameter}" in
    --k8s-version)
      shift
      K8S_VERSION="$1"
      ;;
    --knative-version)
      shift
      KNATIVE_VERSION="$1"
      ;;
    --registry-url)
      shift
      REGISTRY_NAME="$(echo "$1" | cut -d':' -f 1)"
      REGISTRY_PORT="$(echo "$1" | cut -d':' -f 2)"
      ;;
    --cluster-suffix)
      shift
      CLUSTER_SUFFIX="$1"
      ;;
    *) echo "unknown option ${parameter}"; exit 1 ;;
  esac
  shift
done

# The version map correlated with this version of KinD
KIND_VERSION="v0.12.0"
case ${K8S_VERSION} in
  v1.21.x)
    K8S_VERSION="1.21.10"
    KIND_IMAGE_SHA="sha256:84709f09756ba4f863769bdcabe5edafc2ada72d3c8c44d6515fc581b66b029c"
    KIND_IMAGE="kindest/node:v${K8S_VERSION}@${KIND_IMAGE_SHA}"
    ;;
  v1.22.x)
    K8S_VERSION="1.22.7"
    KIND_IMAGE_SHA="sha256:1dfd72d193bf7da64765fd2f2898f78663b9ba366c2aa74be1fd7498a1873166"
    KIND_IMAGE="kindest/node:v${K8S_VERSION}@${KIND_IMAGE_SHA}"
    ;;
  v1.23.x)
    K8S_VERSION="1.23.4"
    KIND_IMAGE_SHA="sha256:0e34f0d0fd448aa2f2819cfd74e99fe5793a6e4938b328f657c8e3f81ee0dfb9"
    KIND_IMAGE="kindest/node:v${K8S_VERSION}@${KIND_IMAGE_SHA}"
    ;;
  v1.24.x)
    KIND_IMAGE_SHA="sha256:0866296e693efe1fed79d5e6c7af8df71fc73ae45e3679af05342239cdc5bc8e"
    KIND_IMAGE=kindest/node:${K8S_VERSION}@${KIND_IMAGE_SHA}
    ;;
  *) echo "Unsupported version: ${K8S_VERSION}"; exit 1 ;;
esac

#############################################################
#
#    Install KinD
#
#############################################################
echo '::group:: Install KinD'

# This does not work on mac, so skip.
if [ ${RUNNING_ON_MAC} == "false" ]; then
  # Disable swap otherwise memory enforcement does not work
  # See: https://kubernetes.slack.com/archives/CEKK1KTN2/p1600009955324200
  sudo swapoff -a
  sudo rm -f /swapfile

  # Use in-memory storage to avoid etcd server timeouts.
  # https://kubernetes.slack.com/archives/CEKK1KTN2/p1615134111016300
  # https://github.com/kubernetes-sigs/kind/issues/845
  if mountpoint -q /tmp/etcd; then
    sudo umount /tmp/etcd
  fi
  sudo mkdir -p /tmp/etcd
  sudo mount -t tmpfs tmpfs /tmp/etcd
fi

if ! command -v kind > /dev/null; then
  curl -Lo ./kind "https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-$(uname)-amd64"
  chmod +x ./kind
  sudo mv kind /usr/local/bin
fi

echo '::endgroup::'


#############################################################
#
#    Setup KinD cluster.
#
#############################################################
echo '::group:: Build KinD Config'

cat > kind.yaml <<EOF
apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
nodes:
- role: control-plane
  image: "${KIND_IMAGE}"
EOF
if [ ${RUNNING_ON_MAC} == "false" ]; then
  cat >> kind.yaml <<EOF_2
  extraMounts:
  - containerPath: /var/lib/etcd
    hostPath: /tmp/etcd
EOF_2
fi
cat >> kind.yaml <<EOF_3
- role: worker
  image: "${KIND_IMAGE}"

# Configure registry for KinD.
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."$REGISTRY_NAME:$REGISTRY_PORT"]
    endpoint = ["http://$REGISTRY_NAME:$REGISTRY_PORT"]

# This is needed in order to support projected volumes with service account tokens.
# See: https://kubernetes.slack.com/archives/CEKK1KTN2/p1600268272383600
kubeadmConfigPatches:
  - |
    apiVersion: kubeadm.k8s.io/v1beta2
    kind: ClusterConfiguration
    metadata:
      name: config
    apiServer:
      extraArgs:
        "service-account-issuer": "https://kubernetes.default.svc"
        "service-account-key-file": "/etc/kubernetes/pki/sa.pub"
        "service-account-signing-key-file": "/etc/kubernetes/pki/sa.key"
        "service-account-api-audiences": "api,spire-server"
        "service-account-jwks-uri": "https://kubernetes.default.svc/openid/v1/jwks"
    networking:
      dnsDomain: "${CLUSTER_SUFFIX}"
EOF_3

cat kind.yaml
echo '::endgroup::'

echo '::group:: Create KinD Cluster'
kind create cluster --config kind.yaml --wait 5m

kubectl describe nodes
echo '::endgroup::'

echo '::group:: Expose OIDC Discovery'

# From: https://banzaicloud.com/blog/kubernetes-oidc/
# To be able to fetch the public keys and validate the JWT tokens against
# the Kubernetes cluster’s issuer we have to allow external unauthenticated
# requests. To do this, we bind this special role with a ClusterRoleBinding
# to unauthenticated users (make sure that this is safe in your environment,
# but only public keys are visible on this URL)
kubectl create clusterrolebinding oidc-reviewer \
  --clusterrole=system:service-account-issuer-discovery \
  --group=system:unauthenticated

echo '::endgroup::'


#############################################################
#
#    Setup metallb
#
#############################################################
echo '::group:: Setup metallb'

kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.9.3/manifests/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.9.3/manifests/metallb.yaml
kubectl create secret generic -n metallb-system memberlist --from-literal=secretkey="$(openssl rand -base64 128)"

network=$(docker network inspect kind -f "{{(index .IPAM.Config 0).Subnet}}" | cut -d '.' -f1,2)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: metallb-system
  name: config
data:
  config: |
    address-pools:
    - name: default
      protocol: layer2
      addresses:
      - $network.255.1-$network.255.250
EOF

echo '::endgroup::'


#############################################################
#
#    Setup container registry
#
#############################################################
echo '::group:: Setup container registry'


docker run -d --restart=always \
       -p "$REGISTRY_PORT:$REGISTRY_PORT" --name "$REGISTRY_NAME" registry:2

# Connect the registry to the KinD network.
docker network connect "kind" "$REGISTRY_NAME"

if ! grep -q "$REGISTRY_NAME" /etc/hosts; then
  # Make the $REGISTRY_NAME -> 127.0.0.1, to tell `ko` to publish to
  # local reigstry, even when pushing $REGISTRY_NAME:$REGISTRY_PORT/some/image
  echo "127.0.0.1 $REGISTRY_NAME" | sudo tee -a /etc/hosts
fi

echo '::endgroup::'


#############################################################
#
#    Install Knative Serving
#
#############################################################
echo '::group:: Install Knative Serving'

# Eliminates the resources blocks in a release yaml
function resource_blaster() {
  local REPO="${1}"
  local FILE="${2}"

  curl -L -s "https://github.com/knative/${REPO}/releases/download/knative-v${KNATIVE_VERSION}/${FILE}" \
    | yq e 'del(.spec.template.spec.containers[]?.resources)' - \
    `# Filter out empty objects that come out as {} b/c kubectl barfs` \
    | grep -v '^{}$'
}

resource_blaster serving serving-crds.yaml | kubectl apply -f -
sleep 3 # Avoid the race creating CRDs then instantiating them...
resource_blaster serving serving-core.yaml | kubectl apply -f -
resource_blaster net-kourier kourier.yaml | kubectl apply -f -
kubectl patch configmap/config-network \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'

# Wait for Knative to be ready (or webhook will reject SaaS)
for x in $(kubectl get deploy --namespace knative-serving -oname); do
  kubectl rollout status --timeout 5m --namespace knative-serving "$x"
done

# Enable the features we need that are currently feature-flagged in Knative.
# We do this last to ensure the webhook is up.
while ! kubectl patch configmap/config-features \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"kubernetes.podspec-fieldref":"enabled", "kubernetes.podspec-volumes-emptydir":"enabled", "multicontainer":"enabled"}}'
do
    echo Waiting for webhook to be up.
    sleep 1
done

# Adjust some default values.
#  - revision-timeout-seconds: reduces the default pod grace period from 5m to 30s
#   (so that things scale down faster).
#  - container-concurrency: sets the default request concurrency to match the default
#   GRPC concurrent streams: https://github.com/grpc/grpc-go/blob/87eb5b7/internal/transport/defaults.go#L34
while ! kubectl patch configmap/config-defaults \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"revision-timeout-seconds":"30","container-concurrency":"100"}}'
do
    echo Waiting for webhook to be up.
    sleep 1
done

# Use min-scale: 1 during tests to preserve logs, use max-scale: 1 to avoid crowding the cluster.
while ! kubectl patch configmap/config-autoscaler \
  --namespace knative-serving \
  --type merge \
  --patch '{"data":{"min-scale":"1","max-scale":"1"}}'
do
    echo Waiting for webhook to be up.
    sleep 1
done

# Enable magic dns so we can interact with minio from our scripts.
resource_blaster serving serving-default-domain.yaml | kubectl apply -f -

# Wait for the job to complete, so we can reliably use ksvc hostnames.
kubectl wait -n knative-serving --timeout=180s --for=condition=Complete jobs --all

echo '::endgroup::'
