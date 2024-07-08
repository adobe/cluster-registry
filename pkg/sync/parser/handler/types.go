package handler

import (
	"context"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TargetObject is an alias for the object type that will be patched with the extracted metadata from the source object
type TargetObject v1.ClusterSpec

type ObjectHandler interface {
	Handle(ctx context.Context, objects []unstructured.Unstructured) ([]byte, error)
}
