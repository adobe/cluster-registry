/*
Copyright 2021 Adobe. All rights reserved.
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
)

// ServiceMetadataWatcherSpec defines the desired state of ServiceMetadataWatcher
type ServiceMetadataWatcherSpec struct {
	WatchedServiceObjects []WatchedServiceObject `json:"watchedServiceObjects"`
}

type WatchedServiceObject struct {
	ObjectReference ObjectReference `json:"objectReference"`
	WatchedFields   []WatchedField  `json:"watchedFields"`
}

type ObjectReference struct {
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

func (o *ObjectReference) String() string {
	return o.Name + "/" + o.APIVersion + "/" + o.Kind
}

type WatchedField struct {
	Source      string `json:"src"`
	Destination string `json:"dst"`
}

type WatchedServiceObjectStatus struct {
	LastUpdated     metav1.Time     `json:"lastUpdated"`
	ObjectReference ObjectReference `json:"objectReference"`
	Errors          []string        `json:"errors"`
}

// ServiceMetadataWatcherStatus defines the observed state of ServiceMetadataWatcher
type ServiceMetadataWatcherStatus struct {
	WatchedServiceObjects []WatchedServiceObjectStatus `json:"watchedServiceObjects"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ServiceMetadataWatcher is the Schema for the servicemetadatawatchers API
type ServiceMetadataWatcher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceMetadataWatcherSpec   `json:"spec,omitempty"`
	Status ServiceMetadataWatcherStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceMetadataWatcherList contains a list of ServiceMetadataWatcher
type ServiceMetadataWatcherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceMetadataWatcher `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceMetadataWatcher{}, &ServiceMetadataWatcherList{})
}
