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

type ObjectHandler interface {
	Handle(ctx context.Context, objects []unstructured.Unstructured) (*v1.ClusterSpec, error)
}
