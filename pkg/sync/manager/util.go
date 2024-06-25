package manager

import (
	"crypto/sha256"
	"fmt"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	jsonpatch "github.com/evanphx/json-patch/v5"
	jsoniter "github.com/json-iterator/go"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type PatchWrapper struct {
	Data map[string]interface{} `json:"data"`
}

func fingerprint(obj interface{}) string {
	b, _ := json.Marshal(obj)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", b)))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func createClusterSpecPatch(clusterSpec *v1.ClusterSpec) ([]byte, error) {
	oldData, err := json.Marshal(v1.ClusterSpec{})
	if err != nil {
		return nil, err
	}
	newData, err := json.Marshal(clusterSpec)
	if err != nil {
		return nil, err
	}
	return jsonpatch.CreateMergePatch(oldData, newData)
}

func mergePatches(a, b []byte) ([]byte, error) {
	if a == nil {
		a = []byte(`{}`)
	}
	if b == nil {
		b = []byte(`{}`)
	}
	return jsonpatch.MergeMergePatches(a, b)
}

func getNestedString(obj unstructured.Unstructured, fields ...string) (string, error) {
	value, found, err := unstructured.NestedString(obj.Object, fields...)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("%s not found in %s object", strings.Join(fields, "."), obj.GroupVersionKind().String())
	}
	return value, nil
}

func getNestedStringSlice(obj unstructured.Unstructured, fields ...string) ([]string, error) {
	value, found, err := unstructured.NestedStringSlice(obj.Object, fields...)
	if err != nil {
		return []string{}, err
	}
	if !found {
		return []string{}, fmt.Errorf("%s not found in %s object", strings.Join(fields, "."), obj.GroupVersionKind().String())
	}
	return value, nil
}
