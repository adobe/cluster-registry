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
	"strings"
)

// AWSManagedControlPlaneHandler extracts the following Cluster Registry metadata from AWSManagedControlPlane (CAPA) objects:
// - extra.oidcIssuer (e.g. "oidc.eks.us-west-2.amazonaws.com/id/EXAMPLED539D4633E2D5B6B716D3041E")
type AWSManagedControlPlaneHandler struct{}

func (h *AWSManagedControlPlaneHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) ([]byte, error) {
	targetObject := new(TargetObject)

	for _, obj := range objects {
		oidcProviderARN, err := getNestedString(obj, "status", "oidcProvider", "arn")
		if err != nil {
			return nil, err
		}
		targetObject.Extra.OidcIssuer = extractOIDCIssuerFromARN(oidcProviderARN)
	}

	return createTargetObjectPatch(targetObject)
}

func extractOIDCIssuerFromARN(arn string) string {
	_, after, found := strings.Cut(arn, "/")
	if !found {
		return ""
	}
	return after
}
