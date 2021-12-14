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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// ClientConfigStatus defines the observed state of ClientConfig
type ClientConfigStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClientConfig is the Schema for the clientconfigs API
type ClientConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the contfigurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	Namespace string `json:"namespace,omitempty"`

	AlertmanagerWebhook AlertmanagerWebhookConfig `json:"alertmanagerWebhook"`
}

// AlertmanagerWebhookConfig ...
type AlertmanagerWebhookConfig struct {
	BindAddress string      `json:"bindAddress"`
	AlertMap    []AlertRule `json:"alertMap"`
}

// AlertRule ...
type AlertRule struct {
	AlertName  string            `json:"alertName"`
	OnFiring   map[string]string `json:"onFiring"`
	OnResolved map[string]string `json:"onResolved"`
}

func init() {
	SchemeBuilder.Register(&ClientConfig{})
}
