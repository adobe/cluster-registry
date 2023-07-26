/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package v2

import (
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
)

type clusterList struct {
	Items      []*registryv1.ClusterSpec `json:"items"`
	ItemsCount int                       `json:"itemsCount"`
	Offset     int                       `json:"offset"`
	Limit      int                       `json:"limit"`
	More       bool                      `json:"more"`
}

type serviceMetadataList struct {
	Items      []*ServiceMetadata `json:"items"`
	ItemsCount int                `json:"itemsCount"`
	Offset     int                `json:"offset"`
	Limit      int                `json:"limit"`
	More       bool               `json:"more"`
}

type ServiceMetadata struct {
	Name            string                     `json:"name"`
	ServiceMetadata registryv1.ServiceMetadata `json:"services"`
}

func newClusterResponse(cluster *registryv1.Cluster) *registryv1.ClusterSpec {
	cs := &cluster.Spec
	cs.ServiceMetadata = nil
	return cs
}

func newClusterListResponse(clusters []registryv1.Cluster, count int, offset int, limit int, more bool) *clusterList {
	r := new(clusterList)
	r.Items = make([]*registryv1.ClusterSpec, 0) // TODO: check memory allocation

	for _, c := range clusters {
		cs := c.Spec
		cs.ServiceMetadata = nil
		r.Items = append(r.Items, &cs)
	}

	r.ItemsCount = count
	r.Offset = offset
	r.Limit = limit
	r.More = more

	return r
}

func newServiceMetadataResponse(cluster *registryv1.Cluster) *ServiceMetadata {
	return &ServiceMetadata{
		Name:            cluster.Spec.Name,
		ServiceMetadata: cluster.Spec.ServiceMetadata,
	}
}

func newServiceMetadataListResponse(clusters []registryv1.Cluster, count int, offset int, limit int, more bool) *serviceMetadataList {
	r := new(serviceMetadataList)
	r.Items = make([]*ServiceMetadata, 0)

	for _, c := range clusters {
		sm := ServiceMetadata{
			Name:            c.Spec.Name,
			ServiceMetadata: c.Spec.ServiceMetadata,
		}
		r.Items = append(r.Items, &sm)
	}

	r.ItemsCount = count
	r.Offset = offset
	r.Limit = limit
	r.More = more

	return r
}
