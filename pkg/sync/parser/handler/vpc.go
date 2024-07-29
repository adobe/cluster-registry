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
)

// VPCHandler extracts the following Cluster Registry metadata from VPC (ACK) objects:
// virtualNetworks:
// - vpcID (e.g. "vpc-0a1b2c3d4e5f6g7h8")
// - cidrBlocks (e.g. ["10.123.0.0/12", "100.64.0.0/16])
// - accountId (e.g. "123456789012")
type VPCHandler struct{}

func (h *VPCHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) (*v1.ClusterSpec, error) {
	clusterSpec := new(v1.ClusterSpec)

	for _, obj := range objects {
		vpcID, err := getNestedString(obj, "status", "vpcID")
		if err != nil {
			return nil, err
		}

		cidrs, err := getNestedStringSlice(obj, "spec", "cidrBlocks")
		if err != nil {
			return nil, err
		}

		clusterSpec.VirtualNetworks = append(clusterSpec.VirtualNetworks, v1.VirtualNetwork{
			ID:    vpcID,
			Cidrs: cidrs,
		})

		// TODO: maybe there's a better object to extract this from
		accountId, err := getNestedString(obj, "status", "ownerID")
		if err != nil {
			return nil, err
		}

		clusterSpec.AccountID = accountId
	}

	return clusterSpec, nil
}
