/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"time"

	creds "github.com/tektoncd/results/pkg/watcher/grpc"
	"github.com/tektoncd/results/pkg/watcher/reconciler"
	"github.com/tektoncd/results/pkg/watcher/reconciler/pipelinerun"
	"github.com/tektoncd/results/pkg/watcher/reconciler/taskrun"
	v1alpha2pb "github.com/tektoncd/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	_ "knative.dev/pkg/system/testing"
)

var (
	apiAddr          = flag.String("api_addr", "localhost:50051", "Address of API server to report to")
	authMode         = flag.String("auth_mode", "", "Authentication mode to use when making requests. If not set, no additional credentials will be used in the request. Valid values: [google]")
	disableCRDUpdate = flag.Bool("disable_crd_update", false, "Disables Tekton CRD annotation update on reconcile.")
)

func main() {
	flag.Parse()
	// TODO: Enable leader election.
	ctx := sharedmain.WithHADisabled(signals.NewContext())

	conn, err := connectToAPIServer(ctx, *apiAddr, *authMode)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	log.Println("connected!")
	defer conn.Close()
	results := v1alpha2pb.NewResultsClient(conn)

	cfg := &reconciler.Config{
		DisableAnnotationUpdate: *disableCRDUpdate,
	}

	restCfg := sharedmain.ParseAndGetConfigOrDie()
	sharedmain.MainWithConfig(injection.WithNamespaceScope(ctx, ""), "watcher", restCfg,
		func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
			return pipelinerun.NewControllerWithConfig(ctx, results, cfg)
		}, func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
			return taskrun.NewControllerWithConfig(ctx, results, cfg)
		},
	)
}

func connectToAPIServer(ctx context.Context, apiAddr string, authMode string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
	}

	// Setup TLS certs to the server. Do this once since this is likely going
	// to be shared in multiple auth modes.
	certs, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("error loading cert pool: %v", err)
	}
	cred := credentials.NewTLS(&tls.Config{
		RootCAs: certs,
	})

	// Add in additional credentials to requests if desired.
	switch authMode {
	case "google":
		opts = append(opts,
			grpc.WithAuthority(apiAddr),
			grpc.WithTransportCredentials(cred),
			grpc.WithDefaultCallOptions(grpc.PerRPCCredentials(creds.Google())),
		)
	case "insecure":
		opts = append(opts, grpc.WithInsecure())
	}

	log.Printf("dialing %s...\n", apiAddr)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return grpc.DialContext(ctx, apiAddr, opts...)
}
