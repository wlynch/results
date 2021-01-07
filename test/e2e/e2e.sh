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

    export KO_DOCKER_REPO=${KO_DOCKER_REPO:-"kind.local"}

    echo "Installing Tekton Pipelines..."
    TEKTON_PIPELINE_CONFIG=${TEKTON_PIPELINE_CONFIG:-"https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml"}
    kubectl apply --filename ${TEKTON_PIPELINE_CONFIG}

    echo "Generating DB secret..."
    # Don't fail if the secret isn't created - this can happen if the secret already exists.
    kubectl create secret generic tekton-results-mysql --namespace="tekton-pipelines" --from-literal=user=root --from-literal=password=$(openssl rand -base64 20) || true

    echo "Generating TLS key pair..."
    set +e
      openssl req -x509 \
        -newkey rsa:4096 \
        -keyout "/tmp/tekton-results-key.pem" \
        -out "/tmp/tekton-results-cert.pem" \
        -days 365 \
        -nodes \
        -subj "/CN=tekton-results-api-service.tekton-pipelines.svc.cluster.local" \
        -addext "subjectAltName = DNS:tekton-results-api-service.tekton-pipelines.svc.cluster.local"

      if [ $? -ne 0 ] ; then
        # LibreSSL didn't support the -addext flag until version 3.1.0 but
        # version 2.8.3 ships with MacOS Big Sur. So let's try a different way...
        echo "Falling back to legacy libressl cert generation"
        openssl req -x509 \
          -verbose \
          -config <(cat /etc/ssl/openssl.cnf <(printf "[SAN]\nsubjectAltName = DNS:tekton-results-api-service.tekton-pipelines.svc.cluster.local")) \
          -extensions SAN \
          -newkey rsa:4096 \
          -keyout "/tmp/tekton-results-key.pem" \
          -out "/tmp/tekton-results-cert.pem" \
          -days 365 \
          -nodes \
          -subj "/CN=tekton-results-api-service.tekton-pipelines.svc.cluster.local"

        if [ $? -ne 0 ] ; then
          echo "There was an error generating certificates"
          exit 1
        fi
      fi
    set -e
    kubectl create secret tls -n tekton-pipelines tekton-results-tls --cert="/tmp/tekton-results-cert.pem" --key="/tmp/tekton-results-key.pem" || true

    echo "Installing Tekton Results..."
    ko apply --filename="${ROOT}/config/"

    echo "Waiting for deployments to be ready..."
    kubectl wait deployment "tekton-results-mysql" --namespace="tekton-pipelines" --for="condition=available" --timeout="60s"
    kubectl wait deployment "tekton-results-api" --namespace="tekton-pipelines" --for="condition=available" --timeout="60s"
    kubectl wait deployment "tekton-results-watcher" --namespace="tekton-pipelines" --for="condition=available" --timeout="60s"

    # Test
    go test --tags=e2e ./test/e2e/...
}

main