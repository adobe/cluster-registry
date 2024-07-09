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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ClusterHandler extracts the following Cluster Registry metadata from Cluster (CAPI) objects:
// - shortName (e.g. "ethos000devva6")
// - region (e.g. "va6")
// - provider (e.g. "eks")
// - location (e.g. "us-west-2")
// - environment (e.g. "dev")
type ClusterHandler struct{}

func (h *ClusterHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) ([]byte, error) {
	targetObject := new(TargetObject)

	for _, obj := range objects {
		clusterShortName, err := getNestedString(obj, "metadata", "labels", "clusterShortName")
		if err != nil {
			return nil, err
		}
		targetObject.ShortName = clusterShortName

		region, err := getNestedString(obj, "metadata", "labels", "locationShortName")
		if err != nil {
			return nil, err
		}
		targetObject.Region = region

		provider, err := getNestedString(obj, "metadata", "labels", "provider")
		if err != nil {
			return nil, err
		}
		targetObject.CloudType = provider

		cloudProviderRegion, err := getNestedString(obj, "metadata", "labels", "location")
		if err != nil {
			return nil, err
		}
		targetObject.CloudProviderRegion = cloudProviderRegion

		environment, err := getNestedString(obj, "metadata", "labels", "environment")
		if err != nil {
			return nil, err
		}
		targetObject.Environment = environment
	}

	return createTargetObjectPatch(targetObject)
}
