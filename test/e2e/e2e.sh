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
    export KO_DOCKER_REPO="kind.local"
    export KIND_CLUSTER_NAME="tekton-results"

    ROOT="$(git rev-parse --show-toplevel)"

    ${ROOT}/test/e2e/00-setup.sh
    ${ROOT}/test/e2e/01-install.sh

    # Test
    go test --tags=e2e ${ROOT}/test/e2e/...
}

main