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
	"strings"
)

// SubnetHandler extracts the following Cluster Registry metadata from Subnet (ACK) objects:
// availabilityZones:
// - name (e.g. "us-east-1c")
// - id (e.g. "use-az2")
type SubnetHandler struct{}

func (h *SubnetHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) ([]byte, error) {
	targetObject := new(TargetObject)

	for _, obj := range objects {
		if !isPrivateSubnet(obj) {
			// to extract az info, we only care about private subnets
			continue
		}

		availabilityZone, err := getNestedString(obj, "spec", "availabilityZone")
		if err != nil {
			return nil, err
		}

		availabilityZoneID, err := getNestedString(obj, "spec", "availabilityZoneID")
		if err != nil {
			return nil, err
		}

		targetObject.AvailabilityZones = append(targetObject.AvailabilityZones, v1.AvailabilityZone{
			Name: availabilityZone,
			ID:   availabilityZoneID,
		})
	}

	return createTargetObjectPatch(targetObject)
}

// isPrivateSubnet returns true if the subnet is private
func isPrivateSubnet(obj unstructured.Unstructured) bool {
	// TODO: ideally the subnet type should be a label or annotation, but until then we can use the name
	return strings.Contains(obj.GetName(), "private")
}
