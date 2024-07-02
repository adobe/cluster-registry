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
