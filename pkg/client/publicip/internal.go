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

package publicip

import (
	"context"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type scanner struct {
	client    client.Client
	logger    logr.Logger
	namespace string
}

func (s *scanner) GetClient() client.Client {
	return s.client
}

func (s *scanner) Run(ctx context.Context) error {
	clusterList := &registryv1.ClusterList{}
	err := s.client.List(context.TODO(), clusterList, &client.ListOptions{Namespace: s.namespace})
	if err != nil {
		return err
	}

	for _, cluster := range clusterList.Items {
		switch cluster.Spec.CloudType {
		case "aws", "eks":
			s.logger.Info("Querying AWS cloud provider API", "cluster", cluster.Name)

		case "azure", "aks":
			s.logger.Info("Querying Azure cloud provider API", "cluster", cluster.Name)

		case "datacenter":
			// not yet implemented
			s.logger.Info("Skipping datacenter cluster", "cluster", cluster.Name)

		default:
			s.logger.Info("Unknown cloud provider", "cluster", cluster.Name)
		}
	}

	return nil
}
