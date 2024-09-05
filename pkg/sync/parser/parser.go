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

package parser

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	registryv1alpha1 "github.com/adobe/cluster-registry/pkg/api/registry/v1alpha1"
	"github.com/adobe/cluster-registry/pkg/sync/parser/handler"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceParser is responsible for parsing any watched resource that has a registered handler
type ResourceParser struct {
	client.Client
	log      logr.Logger
	handlers map[schema.GroupVersionKind]handler.ObjectHandler
	buffer   v1.ClusterSpec
}

func New(client client.Client, log logr.Logger) *ResourceParser {
	return &ResourceParser{
		Client:   client,
		log:      log,
		handlers: make(map[schema.GroupVersionKind]handler.ObjectHandler),
		buffer:   v1.ClusterSpec{},
	}
}

// RegisterHandlerForGVK registers a handler for a specific GroupVersionKind
func (p *ResourceParser) RegisterHandlerForGVK(gvk schema.GroupVersionKind, parser handler.ObjectHandler) {
	p.handlers[gvk] = parser
}

// Parse parses the watched resource and returns a byte array that represents a patch to be applied to the target resource
func (p *ResourceParser) Parse(ctx context.Context, res registryv1alpha1.WatchedResource) error {
	gvk, err := res.GVK()
	if err != nil {
		return err
	}

	h, ok := p.handlers[gvk]
	if !ok {
		return fmt.Errorf("no handler registered for GVK: %s", gvk.String())
	}

	objects, err := p.getObjectsForResource(ctx, res)
	if err != nil {
		return err
	}

	patch, err := h.Handle(ctx, objects)
	if err != nil {
		return err
	}

	return p.buffer.Merge(patch)
}

func (p *ResourceParser) GetBuffer() v1.ClusterSpec {
	return p.buffer
}

func (p *ResourceParser) SetBuffer(buffer v1.ClusterSpec) {
	p.buffer = buffer
}

func (p *ResourceParser) Diff() ([]byte, error) {
	original, err := json.Marshal(v1.ClusterSpec{})
	if err != nil {
		return nil, err
	}

	modified, err := json.Marshal(p.buffer)
	if err != nil {
		return nil, err
	}

	return jsonpatch.CreateMergePatch(original, modified)
}

func (p *ResourceParser) getObjectsForResource(ctx context.Context, res registryv1alpha1.WatchedResource) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	gvk, err := res.GVK()
	if err != nil {
		return objects, err
	}

	log := p.log.WithValues("gvk", gvk, "namespace", res.Namespace)

	if res.Name != "" {
		// get a single object
		p.log.WithValues("name", res.Name)
		obj := new(unstructured.Unstructured)
		obj.SetGroupVersionKind(gvk)
		if err := p.Client.Get(ctx, types.NamespacedName{
			Name:      res.Name,
			Namespace: res.Namespace,
		}, obj); err != nil {
			log.Error(err, "cannot get object")
			return nil, err
		}

		objects = append(objects, *obj)
	} else {
		// get a list of objects
		listOptions := &client.ListOptions{Namespace: res.Namespace}
		if res.LabelSelector != nil {
			selector, err := metav1.LabelSelectorAsSelector(res.LabelSelector)
			if err != nil {
				return nil, err
			}
			listOptions.LabelSelector = selector
			log.WithValues("labelSelector", res.LabelSelector.String())
		}
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := p.Client.List(ctx, list, listOptions); err != nil {
			p.log.Error(err, "cannot list objects")
		}

		objects = append(objects, list.Items...)
	}
	return objects, nil
}
