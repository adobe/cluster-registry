package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WatchedResource struct {
	// Kind of the resource
	Kind string `json:"kind,omitempty"`
	// API version of the resource
	APIVersion string `json:"apiVersion,omitempty"`
	// Namespace of the resource
	Namespace string `json:"namespace,omitempty"`
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
}

// ClusterSyncStatus defines the observed state of ClusterSync
type ClusterSyncStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterSync is the Schema for the ClusterSync API
type ClusterSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterSyncSpec `json:"spec,omitempty"`
	// +optional
	Data   map[string]string `json:"data,omitempty"`
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
