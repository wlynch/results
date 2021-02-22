// Copyright 2021 The Tekton Authors
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

package reconciler

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigMapName = "watcher-config"
)

// Config defines shared reconciler configuration options.
type Config struct {
	// Configures whether Tekton CRD objects should be updated with Result
	// annotations during reconcile. Useful to enable for dry run modes.
	DisableAnnotationUpdate bool
}

// GetDisableAnnotationupdate returns whether annotation updates should be
// disabled. This is safe to call for missing configs.
func (c *Config) GetDisableAnnotationUpdate() bool {
	if c == nil {
		return false
	}
	return c.DisableAnnotationUpdate
}

func ToConfigMap(c *Config) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: ConfigMapName,
		},
		Data: map[string]string{
			"disable_annotation_update": fmt.Sprintf("%t", c.GetDisableAnnotationUpdate()),
		},
	}
}

func FromConfigMap(m *corev1.ConfigMap) *Config {
	if m == nil {
		return nil
	}
	return &Config{
		DisableAnnotationUpdate: strings.EqualFold(m.Data["disable_annotation_update"], "true"),
	}
}
