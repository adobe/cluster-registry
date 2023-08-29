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

package sqs

import (
	"context"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
)

type fakeproducer struct {
}

// NewFakeProducer creates a fake producer
func NewFakeProducer(m monitoring.MetricsI) Producer {
	return &fakeproducer{}
}

// Send message in sqs queue
func (p *fakeproducer) Send(ctx context.Context, c *registryv1.Cluster) error {
	return nil
}
