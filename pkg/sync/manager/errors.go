package manager

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

type FailedToGetValueFromObjectError struct {
	Fields []string
	Object unstructured.Unstructured
	Err    error
}

func (e *FailedToGetValueFromObjectError) Error() string {
	return fmt.Sprintf("failed to get value (%s) from object (namespace: %s, name: %s, gvk: %s): %s",
		strings.Join(e.Fields, "."),
		e.Object.GetNamespace(), e.Object.GetName(), e.Object.GroupVersionKind().String(),
		e.Err.Error())
}

func (e *FailedToGetValueFromObjectError) Wrap(err error) error {
	e.Err = err
	return e
}
