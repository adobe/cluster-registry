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

package v1

import (
	"dario.cat/mergo"
	"reflect"
)

func (dst *ClusterSpec) Merge(src *ClusterSpec) error {
	return mergo.Merge(dst, src, mergo.WithOverride, mergo.WithTransformers(clusterSpecTransformer{}))
}

type clusterSpecTransformer struct{}

func (t clusterSpecTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf([]Tier{}) {
		return func(dst, src reflect.Value) error {
			return t.mergeTiers(dst, src)
		}
	}
	return nil
}

func (t clusterSpecTransformer) mergeTiers(dst, src reflect.Value) error {
	if !dst.CanSet() || !src.CanSet() {
		return nil
	}

	srcs := src.Interface().([]Tier)
	dsts := dst.Interface().([]Tier)

	for _, s := range srcs {
		di := -1
		for i, d := range dsts {
			if d.Name == s.Name {
				di = i
				break
			}
		}
		if di < 0 {
			continue
		}

		err := mergo.Merge(&dsts[di], s, mergo.WithAppendSlice, mergo.WithSliceDeepCopy, mergo.WithTransformers(&clusterSpecTransformer{}))
		if err != nil {
			return err
		}
	}

	dst.Set(reflect.ValueOf(dsts))
	return nil
}
