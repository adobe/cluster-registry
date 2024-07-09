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
	"encoding/json"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func createTargetObjectPatch(targetObject *TargetObject) ([]byte, error) {
	oldData, err := json.Marshal(TargetObject{})
	if err != nil {
		return nil, err
	}
	newData, err := json.Marshal(targetObject)
	if err != nil {
		return nil, err
	}
	return jsonpatch.CreateMergePatch(oldData, newData)
}

func getNestedString(obj unstructured.Unstructured, fields ...string) (string, error) {
	value, found, err := unstructured.NestedString(obj.Object, fields...)
	getErr := &ParseError{
		Fields: fields,
		Object: obj,
	}
	if err != nil {
		return "", getErr.Wrap(err)
	}
	if !found {
		return "", getErr.Wrap(fmt.Errorf("field not found"))
	}
	return value, nil
}

func getNestedStringSlice(obj unstructured.Unstructured, fields ...string) ([]string, error) {
	value, found, err := unstructured.NestedStringSlice(obj.Object, fields...)
	getErr := &ParseError{
		Fields: fields,
		Object: obj,
	}
	if err != nil {
		return []string{}, getErr.Wrap(err)
	}
	if !found {
		return []string{}, getErr.Wrap(fmt.Errorf("field not found"))
	}
	return value, nil
}
