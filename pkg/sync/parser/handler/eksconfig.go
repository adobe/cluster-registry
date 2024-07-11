/*
Copyright 2024 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package handler

import (
	"context"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"slices"
)

// EKSConfigHandler extracts the following Cluster Registry metadata from EKSConfig (CAPA) objects:
// - tiers[].containerRuntime (e.g. "containerd")
type EKSConfigHandler struct{}

func (h *EKSConfigHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) (*v1.ClusterSpec, error) {
	clusterSpec := new(v1.ClusterSpec)

	for _, obj := range objects {
		containerRuntime, err := getNestedString(obj, "spec", "containerRuntime")
		if err != nil {
			return nil, err
		}

		if k := slices.IndexFunc(clusterSpec.Tiers, func(t v1.Tier) bool {
			return t.Name == obj.GetName()
		}); k > -1 {
			clusterSpec.Tiers[k].ContainerRuntime = containerRuntime
		} else {
			clusterSpec.Tiers = append(clusterSpec.Tiers, v1.Tier{
				Name:             obj.GetName(),
				ContainerRuntime: containerRuntime,
			})
		}
	}

	return clusterSpec, nil
}
