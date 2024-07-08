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
	"crypto/sha256"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type PatchWrapper struct {
	Data map[string]interface{} `json:"data"`
}

func hash(obj interface{}) string {
	b, _ := json.Marshal(obj)
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", b)))
	return fmt.Sprintf("%x", h.Sum(nil))
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
