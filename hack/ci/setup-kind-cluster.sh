#!/usr/bin/env bash

# Copyright 2022 The Operating System Manager contributors.
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

set -euo pipefail
set -x

source hack/lib.sh

echodate "Setting up kind cluster..."

if [ -z "${JOB_NAME:-}" ] || [ -z "${PROW_JOB_ID:-}" ]; then
  echodate "This script should only be running in a CI environment."
  exit 1
fi

export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-osm}"

start_docker_daemon_ci

# Make debugging a bit better
echodate "Configuring bash"
cat << EOF >> ~/.bashrc
# Gets set to the CI clusters kubeconfig from a preset
unset KUBECONFIG

cn() {
  kubectl config set-context --current --namespace=\$1
}

kubeconfig() {
  TMP_KUBECONFIG=\$(mktemp);
  kubectl get secret admin-kubeconfig -o go-template='{{ index .data "kubeconfig" }}' | base64 -d > \$TMP_KUBECONFIG;
  export KUBECONFIG=\$TMP_KUBECONFIG;
  cn kube-system
}

# this alias makes it so that watch can be used with other aliases, like "watch k get pods"
alias watch='watch '
alias k=kubectl
alias ll='ls -lh --file-type --group-directories-first'
alias lll='ls -lahF --group-directories-first'
source <(k completion bash )
source <(k completion bash | sed s/kubectl/k/g)
EOF

# Create kind cluster
TEST_NAME="Create kind cluster"
echodate "Creating the kind cluster"
export KUBECONFIG=~/.kube/config

beforeKindCreate=$(nowms)

if [ -n "${DOCKER_REGISTRY_MIRROR_ADDR:-}" ]; then
  mirrorHost="$(echo "$DOCKER_REGISTRY_MIRROR_ADDR" | sed 's#http://##' | sed 's#/+$##g')"

  # make the registry mirror available as a socket,
  # so we can mount it into the kind cluster
  mkdir -p /mirror
  socat UNIX-LISTEN:/mirror/mirror.sock,fork,reuseaddr,unlink-early,mode=777 TCP4:$mirrorHost &

  cat << EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: "${KIND_CLUSTER_NAME}"
nodes:
  - role: control-plane
    # mount the socket
    extraMounts:
    - hostPath: /mirror
      containerPath: /mirror
containerdConfigPatches:
  # point to the soon-to-start local socat process
  - |-
    [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["http://127.0.0.1:5001"]
EOF

  kind create cluster --config kind-config.yaml
  pushElapsed kind_cluster_create_duration_milliseconds $beforeKindCreate

  # unwrap the socket inside the kind cluster and make it available on a TCP port,
  # because containerd/Docker doesn't support sockets for mirrors.
  docker exec kubermatic-control-plane bash -c 'socat TCP4-LISTEN:5001,fork,reuseaddr UNIX:/mirror/mirror.sock &'
else
  kind create cluster --name "$KIND_CLUSTER_NAME"
fi

echodate "Kind cluster $KIND_CLUSTER_NAME is up and running."

