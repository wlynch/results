#!/bin/bash

# standard bash error handling
set -o errexit;
set -o pipefail;
set -o nounset;
# debug commands
set -x;

# cleanup on exit (useful for running locally)
cleanup() {
    kind delete cluster || true
}
trap cleanup EXIT

main() {
    ROOT="$(git rev-parse --show-toplevel)"

    # Setup
    kind create cluster --loglevel=debug --config "${ROOT}/test/e2e/kind-cluster.yaml"
    kind export kubeconfig

    # Install

    # TODO: ko fails if we try to run/source install.sh, but doesn't if we inline verbatim.
    # We should figure out how we can depend on the same script - there's probably a load bearing
    # local variable somewhere.

    export KO_DOCKER_REPO="kind.local"
    export KIND_CLUSTER_NAME="kind"

    ./test/e2e/01-install.sh

    # Test
    go test --tags=e2e ./test/e2e/...
}

main