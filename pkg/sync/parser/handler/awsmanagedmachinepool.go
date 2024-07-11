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
	"encoding/json"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AWSManagedMachinePoolHandler extracts the following Cluster Registry metadata from AWSManagedMachinePoolHandler (CAPA) objects:
// tiers:
// - name (e.g. "worker0")
// - instanceType (e.g. "m5.large")
// - minCapacity (e.g. 1)
// - maxCapacity (e.g. 10)
// - labels (e.g. {"tier": "worker"})
// - taints (e.g. ["node-role.kubernetes.io/worker:NoSchedule"])
type AWSManagedMachinePoolHandler struct{}

func (h *AWSManagedMachinePoolHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) (*v1.ClusterSpec, error) {
	clusterSpec := new(v1.ClusterSpec)

	for _, obj := range objects {
		name, err := getNestedString(obj, "spec", "awsLaunchTemplate", "name")
		if err != nil {
			return nil, err
		}

		instanceType, err := getNestedString(obj, "spec", "awsLaunchTemplate", "instanceType")
		if err != nil {
			return nil, err
		}

		minSize, err := getNestedInt64(obj, "spec", "scaling", "minSize")
		if err != nil {
			return nil, err
		}

		maxSize, err := getNestedInt64(obj, "spec", "scaling", "maxSize")
		if err != nil {
			return nil, err
		}

		labels, err := getNestedStringMap(obj, "spec", "labels")
		if err != nil {
			return nil, err
		}

		taints, err := getTaintsAsStringSlice(obj, "spec", "taints")
		if err != nil {
			return nil, err
		}

		tier := v1.Tier{
			Name:         name,
			InstanceType: instanceType,
			MinCapacity:  int(minSize),
			MaxCapacity:  int(maxSize),
			Labels:       labels,
			Taints:       taints,
		}

		if k := slices.IndexFunc(clusterSpec.Tiers, func(t v1.Tier) bool {
			return t.Name == name
		}); k > -1 {
			clusterSpec.Tiers[k] = tier
		} else {
			clusterSpec.Tiers = append(clusterSpec.Tiers, tier)
		}
	}

	return clusterSpec, nil
}

func getTaintsAsStringSlice(obj unstructured.Unstructured, fields ...string) ([]string, error) {
	taints, err := getNestedSlice(obj, fields...)
	if err != nil {
		return nil, err
	}

	var taintsSlice []string
	for _, t := range taints {
		taint := corev1.Taint{}
		taintJson, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(taintJson, &taint)
		if err != nil {
			return nil, err
		}
		taintsSlice = append(taintsSlice, taint.ToString())
	}

	return taintsSlice, nil
}
