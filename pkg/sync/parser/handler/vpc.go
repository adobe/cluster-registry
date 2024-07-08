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
type VPCHandler struct{}

func (h *VPCHandler) Handle(ctx context.Context, objects []unstructured.Unstructured) ([]byte, error) {
	targetObject := new(TargetObject)

	for _, obj := range objects {
		vpcID, err := getNestedString(obj, "status", "vpcID")
		if err != nil {
			return nil, err
		}

		cidrs, err := getNestedStringSlice(obj, "spec", "cidrBlocks")
		if err != nil {
			return nil, err
		}

		targetObject.VirtualNetworks = append(targetObject.VirtualNetworks, v1.VirtualNetwork{
			ID:    vpcID,
			Cidrs: cidrs,
		})
	}

	return createTargetObjectPatch(targetObject)
}
