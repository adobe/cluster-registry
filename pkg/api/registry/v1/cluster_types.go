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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {

	// Cluster name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=3
	Name string `json:"name"`

	// Cluster name, without dash
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:MinLength=3
	ShortName string `json:"shortName"`

	// Information about K8s API endpoint and CA cert
	// +kubebuilder:validation:Required
	APIServer APIServer `json:"apiServer"`

	// Cluster internal region name
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// The cloud provider
	// +kubebuilder:validation:Required
	CloudType string `json:"cloudType"`

	// The cloud provider standard region
	// +kubebuilder:validation:Required
	CloudProviderRegion string `json:"cloudProviderRegion"`

	// Cluster environment
	// +kubebuilder:validation:Required
	Environment string `json:"environment"`

	// The BU that owns the cluster
	// +kubebuilder:validation:Required
	BusinessUnit string `json:"businessUnit"`

	// The BU responsible for paying for the cluster.
	ChargebackBusinessUnit string `json:"chargebackBusinessUnit,omitempty"`

	// Whether the cluster is charged back to the chargebackBusinessUnit
	ChargedBack *bool `json:"chargedBack,omitempty"`

	// The Org that is responsible for the cluster operations
	// +kubebuilder:validation:Required
	ManagingOrg string `json:"managingOrg"`

	// The Offering that the cluster is meant for
	// +kubebuilder:validation:Required
	Offering []Offering `json:"offering"`

	// The cloud account associated with the cluster
	// +kubebuilder:validation:Required
	AccountID string `json:"accountId"`

	// List of tiers with their associated information
	// +kubebuilder:validation:Required
	Tiers []Tier `json:"tiers"`

	// Virtual Private Networks information
	// +kubebuilder:validation:Required
	VirtualNetworks []VirtualNetwork `json:"virtualNetworks"`

	// Timestamp when cluster was registered in Cluster Registry
	// +kubebuilder:validation:Required
	RegisteredAt string `json:"registeredAt"`

	// Cluster status
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Inactive;Active;Deprecated;Deleted
	Status string `json:"status"`

	// Cluster phase
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Building;Testing;Running;Upgrading
	Phase string `json:"phase"`

	// Cluster maintenance group
	// +kubebuilder:validation:Required
	MaintenanceGroup string `json:"maintenanceGroup"`

	// The corresponding Argo instance of the cluster
	// +kubebuilder:validation:Required
	ArgoInstance string `json:"argoInstance"`

	// The type of the cluster
	Type string `json:"type,omitempty"`

	// Extra information, not necessary related to the cluster
	Extra Extra `json:"extra,omitempty"`

	// Git teams and/or LDAP groups that are allowed to onboard and deploy on the cluster
	AllowedOnboardingTeams []AllowedOnboardingTeam `json:"allowedOnboardingTeams,omitempty"`

	// List of cluster capabilities
	Capabilities []string `json:"capabilities,omitempty"`

	// Information about Virtual Networks manual peered with the cluster
	PeerVirtualNetworks []PeerVirtualNetwork `json:"peerVirtualNetworks,omitempty"`

	// Timestamp when cluster information was updated
	LastUpdated string `json:"lastUpdated"`

	// Cluster tags that were applied
	Tags map[string]string `json:"tags,omitempty"`

	// Capacity cluster information
	Capacity Capacity `json:"capacity,omitempty"`

	// ServiceMetadata service specific metadata
	ServiceMetadata ServiceMetadata `json:"services,omitempty"`

	// AvailabilityZones cluster availability zones
	AvailabilityZones []AvailabilityZone `json:"availabilityZones,omitempty"`
}

// Offering the cluster is meant for
// +kubebuilder:validation:Enum=CaaS;PaaS
type Offering string

// APIServer - information about K8s API server
type APIServer struct {

	// Information about K8s Api Endpoint
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`

	// Information about K8s Api CA Cert
	CertificateAuthorityData string `json:"certificateAuthorityData"`
}

// AllowedOnboardingTeam represents the Git teams and/or LDAP groups that are allowed to onboard
type AllowedOnboardingTeam struct {

	// Name of the team
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// List of git teams
	GitTeams []string `json:"gitTeams,omitempty"`

	// List of ldap groups
	LdapGroups []string `json:"ldapGroups,omitempty"`
}

// Extra information
type Extra struct {
	// Name of the domain
	DomainName string `json:"domainName"`

	// Load balancer endpoints
	LbEndpoints map[string]string `json:"lbEndpoints"`

	// Logging endpoints
	LoggingEndpoints []map[string]string `json:"loggingEndpoints,omitempty"`

	// List of IAM Arns
	EcrIamArns map[string][]string `json:"ecrIamArns,omitempty"`

	// Egress ports allowed outside of the namespace
	EgressPorts string `json:"egressPorts,omitempty"`

	// NFS information
	NFSInfo []map[string]string `json:"nfsInfo,omitempty"`

	// ExtendedRegion information
	ExtendedRegion string `json:"extendedRegion,omitempty"`

	// OIDC Issuer URL
	OidcIssuer string `json:"oidcIssuer,omitempty"`

	// Namespace Profile Infrastructure Type
	NamespaceProfileInfraType string `json:"namespaceProfileInfraType,omitempty"`
}

// Tier details
type Tier struct {

	// Name of the tier
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the instances
	// +kubebuilder:validation:Required
	InstanceType string `json:"instanceType"`

	// Container runtime
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=docker;cri-o
	ContainerRuntime string `json:"containerRuntime"`

	// Min number of instances
	// +kubebuilder:validation:Required
	MinCapacity int `json:"minCapacity"`

	// Max number of instances
	// +kubebuilder:validation:Required
	MaxCapacity int `json:"maxCapacity"`

	// Instance K8s labels
	Labels map[string]string `json:"labels,omitempty"`

	// Instance K8s taints
	Taints []string `json:"taints,omitempty"`

	// EnableKataSupport
	EnableKataSupport bool `json:"enableKataSupport,omitempty"`

	// KernelParameters
	KernelParameters map[string]string `json:"kernelParameters,omitempty"`
}

// VirtualNetwork information
type VirtualNetwork struct {

	// Virtual private network Id
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// CIDRs used in this VirtualNetwork
	// +kubebuilder:validation:Required
	Cidrs []string `json:"cidrs"`
}

// PeerVirtualNetwork -  peering information done at cluster onboarding
type PeerVirtualNetwork struct {

	// Remote Virtual Netowrk ID
	ID string `json:"id,omitempty"`

	// Remote Virtual Netowrk CIDRs
	Cidrs []string `json:"cidrs,omitempty"`

	// Cloud account of the owner
	OwnerID string `json:"ownerID,omitempty"`
}

// Capacity cluster information
type Capacity struct {
	LastUpdated         string `json:"lastUpdated"`
	ClusterCapacity     int    `json:"clusterCapacity"`
	ClusterProvisioning int    `json:"clusterProvisioning"`
	MaxBQUPerRequest    int    `json:"maxBquPerRequest"`
	ClusterMaxBQU       int    `json:"clusterMaxBqu"`
	ClusterCurrentBQU   int    `json:"clusterCurrentBqu"`
}

type ServiceMetadata map[string]ServiceMetadataItem

type ServiceMetadataItem map[string]ServiceMetadataMap

type ServiceMetadataMap map[string]string

type AvailabilityZone struct {
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// Send/Receive Errors
	// Last Update Timestamp?
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
