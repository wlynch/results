FROM golang:1.14.2-alpine

WORKDIR /kind

# Since go modules isn't enabled by default.
ENV GO111MODULE=on

# Build static binaries; otherwise go test complains.
ENV CGO_ENABLED=0

# We're running tests within kind v0.10.0 clusters.
RUN go get github.com/google/ko/cmd/ko@v0.6.2

RUN apk add --no-cache \
  bash curl docker git jq openssl

# Install kubectl and make sure it's available in the PATH.
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && chmod +x ./kubectl && mv ./kubectl /bin

RUN curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.10.0/kind-$(uname)-amd64" && chmod +x ./kind && mv ./kind /bin