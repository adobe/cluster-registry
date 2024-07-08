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
		clusterShortName, err := getNestedString(obj, "metadata", "labels", "shortName")
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
