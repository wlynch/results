// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package taskrun

import (
	"context"
	"time"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/results/pkg/watcher/reconciler"
	"github.com/tektoncd/results/pkg/watcher/reconciler/annotation"
	"github.com/tektoncd/results/pkg/watcher/results"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/logging"
)

type Reconciler struct {
	client            *results.Client
	pipelineclientset versioned.Interface
	cfg               *reconciler.Config
	enqueue           func(types.NamespacedName, time.Duration)
}

func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	log := logging.FromContext(ctx)
	log.With(zap.String("key", key))

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Lookup current TaskRun.
	tr, err := r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Warnf("TaskRun not found: %v", err)
		return nil
	}
	if err != nil {
		log.Errorf("TaskRun.Get: %v", err)
		return err
	}

	// Update record.
	result, record, err := r.client.Put(ctx, tr)
	if err != nil {
		log.Errorf("error updating Record: %v", err)
		return err
	}

	if a := tr.GetAnnotations(); !r.cfg.GetDisableAnnotationUpdate() && (result.GetName() != a[annotation.Result] || record.GetName() != a[annotation.Record]) {
		// Since at least 1 is missing, update TaskRun with Result Annotations.
		patch, err := annotation.Add(result.GetName(), record.GetName())
		if err != nil {
			log.Errorf("error adding Result annotations: %v", err)
			return err
		}
		if _, err := r.pipelineclientset.TektonV1beta1().TaskRuns(tr.GetNamespace()).Patch(tr.Name, types.MergePatchType, patch); err != nil {
			log.Errorf("TaskRun.Patch: %v", err)
			return err
		}
	}

	inPipeline, _, _ := tr.IsPartOfPipeline()
	// If the TaskRun is complete and not yet marked for deletion, cleanup the
	// run resource from the cluster.
	if tr.IsDone() && r.cfg.GetCompletedResourceGracePeriod() != 0 && !inPipeline {
		// We haven't hit the grace period yet - reenqueue the key for processing later.
		if s := time.Since(record.GetUpdatedTime().AsTime()); s < r.cfg.GetCompletedResourceGracePeriod() {
			log.Infof("taskrun is not ready for deletion - grace period: %v, time since completion: %v", r.cfg.GetCompletedResourceGracePeriod(), s)
			r.enqueue(tr.GetNamespacedName(), r.cfg.GetCompletedResourceGracePeriod())
			return nil
		}

		log.Infof("deleting TaskRun UID %s", tr.GetUID())
		if err := r.pipelineclientset.TektonV1beta1().TaskRuns(tr.GetNamespace()).Delete(tr.Name, &metav1.DeleteOptions{}); err != nil {
			log.Errorf("TaskRun.Delete: %v", err)
			return err
		}
	}
	return nil
}
