#!/bin/bash

export KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-"tekton-results"}
WORKDIR="$(git rev-parse --show-toplevel)/test/e2e"

${WORKDIR}/00-setup.sh

kind export kubeconfig --name=${KIND_CLUSTER_NAME}

${WORKDIR}/01-install.sh || exit 1

${WORKDIR}/02-test.sh