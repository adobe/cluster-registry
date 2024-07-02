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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type WatchedResource struct {
	// Kind of the resource
	Kind string `json:"kind"`
	// API version of the resource
	APIVersion string `json:"apiVersion"`
	// Namespace of the resource
	Namespace string `json:"namespace"`
	// Name of the resource
	// +optional
	Name string `json:"name,omitempty"`
	// Label selector for the resource
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// ClusterSyncSpec defines the desired state of ClusterSync
type ClusterSyncSpec struct {
	// +required
	// +kubebuilder:validation:Required
	WatchedResources []WatchedResource `json:"watchedResources"`
	// +optional
	InitialData string `json:"initialData,omitempty"`
}

// ClusterSyncStatus defines the observed state of ClusterSync
type ClusterSyncStatus struct {
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
	// +optional
	LastSyncStatus *string `json:"lastSyncStatus,omitempty"`
	// +optional
	LastSyncError *string `json:"lastSyncError,omitempty"`
	// +optional
	SyncedData *string `json:"syncedData,omitempty"`
	// +optional
	SyncedDataHash *string `json:"syncedDataHash,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterSync is the Schema for the ClusterSync API
type ClusterSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSyncSpec `json:"spec,omitempty"`
	// +optional
	Status ClusterSyncStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterSyncList contains a list of ClusterSync
type ClusterSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterSync{}, &ClusterSyncList{})
}

func (res *WatchedResource) GVK() (schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(res.APIVersion)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    res.Kind,
	}, nil
}
