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

package database

import (
	"fmt"
	"github.com/adobe/cluster-registry/pkg/apiserver/models"
	"k8s.io/utils/ptr"
	"sort"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/labstack/gommon/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database Suite", func() {
	var db Db
	var appConfig *config.AppConfig
	var m *monitoring.Metrics

	BeforeEach(func() {
		appConfig = &config.AppConfig{
			AwsRegion:   dbTestConfig["AWS_REGION"],
			DbEndpoint:  dbContainer.Endpoint,
			DbTableName: dbTestConfig["DB_TABLE_NAME"],
			DbIndexName: dbTestConfig["DB_INDEX_NAME"],
		}
		m = monitoring.NewMetrics("cluster_registry_api_database_test", true)
		db = NewDb(appConfig, m)

		err := createTable(dbContainer.Endpoint)
		if err != nil {
			log.Fatalf("Error while creating table: %v", err.Error())
		}

		err = importData(db)
		if err != nil {
			log.Fatalf("Error while populating database: %v", err.Error())
		}
	})

	AfterEach(func() {
		err := deleteTable(dbContainer.Endpoint, appConfig.DbTableName)
		if err != nil {
			log.Fatalf("Error while deleting table: %v", err.Error())
		}
	})

	Context("Database tests", func() {

		It("Should handle DB status OK", func() {
			err := db.Status()
			Expect(err).To(BeNil())
		})

		It("Should handle DB status not OK", func() {
			appConfig := &config.AppConfig{
				AwsRegion:   appConfig.AwsRegion,
				DbEndpoint:  dbContainer.Endpoint,
				DbTableName: "wrong-table-names",
				DbIndexName: appConfig.DbIndexName,
			}
			newM := monitoring.NewMetrics("cluster_registry_api_database_test_new", true)
			newDb := NewDb(appConfig, newM)
			err := newDb.Status()
			Expect(err.Error()).To(ContainSubstring("Cannot do operations on a non-existent table"))
		})

		It("Should handle DB Get cluster", func() {
			tcs := []struct {
				name            string
				clusterName     string
				expectedCluster *registryv1.Cluster
			}{
				{
					name:        "existing cluster",
					clusterName: "cluster01-prod-useast1",
					expectedCluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster01-prod-useast1",
							ShortName: "cluster01produseast1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster01-prod-useast1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
							},
							ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
							Region:                 "useast1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-555555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints:            []string{"workload=memory-optimized:NoSchedule"},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
									Cidrs: []string{"10.0.0.0/24"},
								},
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
									Cidrs: []string{"10.1.0.0/24"},
								},
							},
							RegisteredAt:     "2021-12-13T05:50:07.492Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Shared",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster01-prod-useast1.example.com",
								},
								LoggingEndpoints: []map[string]string{
									{
										"region":    "useast1",
										"endpoint":  "splunk-us-east1.example.com",
										"isDefault": "true",
									},
									{
										"isDefault": "false",
										"region":    "useast2",
										"endpoint":  "splunk-us-east2.example.com",
									},
								},
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
										"arn:aws:iam::account-id:role/yyy",
									},
									"iamUser": {
										"arn:aws:iam::111222333:user/ecr-login",
									},
								},
								EgressPorts: "1024-65535",
								NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
							},
							AllowedOnboardingTeams: nil,
							Capabilities:           []string{"vpc-peering", "gpu-compute"},
							PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
								{
									ID:      "123r",
									Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
									OwnerID: "ownerxxx",
								},
							},
							LastUpdated: "2021-12-13T05:50:07.492Z",
							Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
						},
					},
				},
				{
					name:            "non existing cluster",
					clusterName:     "cluster100-prod-useast1",
					expectedCluster: nil,
				},
			}

			for _, tc := range tcs {
				By(fmt.Sprintf("TestCase %s:\t When getting cluster %s", tc.name, tc.clusterName))
				c, err := db.GetCluster(tc.clusterName)

				Expect(err).To(BeNil())
				if tc.expectedCluster == nil {
					Expect(c).To(BeNil())
				} else {
					Expect(c.Spec).To(Equal(tc.expectedCluster.Spec))
				}
			}
		})

		It("Should handle DB Put cluster", func() {
			tcs := []struct {
				name            string
				clusterName     string
				newCluster      *registryv1.Cluster
				expectedCluster *registryv1.Cluster
			}{
				{
					name:        "update existing cluster",
					clusterName: "cluster01-prod-useast1",
					newCluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:        "cluster01-prod-useast1",
							Region:      "useast1",
							Environment: "Prod",
							Offering:    []registryv1.Offering{"caas", "paas"},
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									MinCapacity:       0,
									MaxCapacity:       0,
									EnableKataSupport: false,
								},
								{
									Name:        "workerMemoryOptimized",
									MinCapacity: 0,
									MaxCapacity: 0,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
								},
							},
							Status:       "Active",
							Phase:        "Running",
							Type:         "Restricted",
							Capabilities: []string{"gpu-compute"},
							RegisteredAt: "2022-03-20T07:55:46.132Z",
							LastUpdated:  "2022-03-20T07:55:46.132Z",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
					expectedCluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:        "cluster01-prod-useast1",
							Region:      "useast1",
							Environment: "Prod",
							Offering:    []registryv1.Offering{"caas", "paas"},
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									MinCapacity:       0,
									MaxCapacity:       0,
									EnableKataSupport: false,
								},
								{
									Name:        "workerMemoryOptimized",
									MinCapacity: 0,
									MaxCapacity: 0,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
								},
							},
							Status:       "Active",
							Phase:        "Running",
							Type:         "Restricted",
							Capabilities: []string{"gpu-compute"},
							RegisteredAt: "2021-12-13T05:50:07.492Z", // once the cluster is first registered, this filed cannot be changed
							LastUpdated:  "2022-03-20T07:55:46.132Z",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				},
				{
					name:        "update non existing cluster",
					clusterName: "cluster101-prod-useast1",
					newCluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:        "cluster101-prod-useast1",
							Region:      "useast1",
							Environment: "Prod",
							Offering:    []registryv1.Offering{"caas", "paas"},
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									MinCapacity:       0,
									MaxCapacity:       0,
									EnableKataSupport: false,
								},
								{
									Name:        "workerMemoryOptimized",
									MinCapacity: 0,
									MaxCapacity: 0,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
								},
							},
							Status:       "Active",
							Phase:        "Running",
							Type:         "Restricted",
							Capabilities: []string{"gpu-compute"},
							LastUpdated:  "2020-03-20T07:55:46.132Z",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
					expectedCluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:        "cluster101-prod-useast1",
							Region:      "useast1",
							Environment: "Prod",
							Offering:    []registryv1.Offering{"caas", "paas"},
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									MinCapacity:       0,
									MaxCapacity:       0,
									EnableKataSupport: false,
								},
								{
									Name:        "workerMemoryOptimized",
									MinCapacity: 0,
									MaxCapacity: 0,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
								},
							},
							Status:       "Active",
							Phase:        "Running",
							Type:         "Restricted",
							Capabilities: []string{"gpu-compute"},
							LastUpdated:  "2020-03-20T07:55:46.132Z",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				},
			}

			for _, tc := range tcs {
				By(fmt.Sprintf("TestCase %s:\t When put cluster %s", tc.name, tc.clusterName))

				err := db.PutCluster(tc.newCluster)
				Expect(err).To(BeNil())

				c, err := db.GetCluster(tc.clusterName)
				Expect(err).To(BeNil())

				Expect(c.Spec).To(Equal(tc.expectedCluster.Spec))
			}
		})

		It("Should handle DB Delete cluster", func() {
			tcs := []struct {
				name          string
				clusterName   string
				expectedError error
			}{
				{
					name:          "existing cluster",
					clusterName:   "cluster01-prod-useast1",
					expectedError: nil,
				},
				{
					name:          "non existing cluster",
					clusterName:   "cluster102-prod-useast1",
					expectedError: fmt.Errorf("cluster not found"),
				},
			}

			for _, tc := range tcs {
				By(fmt.Sprintf("TestCase %s:\t When deleting cluster %s", tc.name, tc.clusterName))

				err := db.DeleteCluster(tc.clusterName)
				Expect(err).To(BeNil())

				c, err := db.GetCluster(tc.clusterName)
				Expect(err).To(BeNil())
				Expect(c).To(BeNil())
			}
		})

		It("Should handle DB List clusters", func() {
			tcs := []struct {
				name             string
				queryParams      map[string]string
				offset           int
				limit            int
				expectedCount    int
				expectedMore     bool
				expectedError    error
				expectedClusters []registryv1.Cluster
			}{
				{
					name: "all clusters",
					queryParams: map[string]string{
						"region":      "",
						"environment": "",
						"status":      "",
						"lastUpdated": "",
					},
					offset:        0,
					limit:         10,
					expectedCount: 3,
					expectedMore:  false,
					expectedError: nil,
					expectedClusters: []registryv1.Cluster{
						{
							Spec: registryv1.ClusterSpec{
								Name:      "cluster01-prod-useast1",
								ShortName: "cluster01produseast1",
								APIServer: registryv1.APIServer{
									Endpoint:                 "https://cluster01-prod-useast1.example.com",
									CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
								},
								ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
								Region:                 "useast1",
								CloudType:              "azure",
								Environment:            "Prod",
								BusinessUnit:           "BU1",
								ChargebackBusinessUnit: "BU1",
								ChargedBack:            ptr.To(true),
								Offering:               []registryv1.Offering{"caas", "paas"},
								AccountID:              "11111-2222-3333-4444-555555555",
								Tiers: []registryv1.Tier{
									{
										Name:              "worker",
										InstanceType:      "c5.9xlarge",
										MinCapacity:       3,
										MaxCapacity:       1000,
										EnableKataSupport: false,
									},
									{
										Name:         "workerMemoryOptimized",
										InstanceType: "r5.8xlarge",
										MinCapacity:  0,
										MaxCapacity:  100,
										Labels: map[string]string{
											"node.kubernetes.io/workload.memory-optimized": "true",
										},
										Taints:            []string{"workload=memory-optimized:NoSchedule"},
										EnableKataSupport: false,
									},
								},
								VirtualNetworks: []registryv1.VirtualNetwork{
									{
										ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
										Cidrs: []string{"10.0.0.0/24"},
									},
									{
										ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
										Cidrs: []string{"10.1.0.0/24"},
									},
								},
								RegisteredAt:     "2021-12-13T05:50:07.492Z",
								Status:           "Active",
								Phase:            "Running",
								Type:             "Shared",
								MaintenanceGroup: "B",
								Extra: registryv1.Extra{
									DomainName: "example.com",
									LbEndpoints: map[string]string{
										"public": "cluster01-prod-useast1.example.com",
									},
									LoggingEndpoints: []map[string]string{
										{
											"region":    "useast1",
											"endpoint":  "splunk-us-east1.example.com",
											"isDefault": "true",
										},
										{
											"isDefault": "false",
											"region":    "useast2",
											"endpoint":  "splunk-us-east2.example.com",
										},
									},
									EcrIamArns: map[string][]string{
										"iamRoles": {
											"arn:aws:iam::account-id:role/xxx",
											"arn:aws:iam::account-id:role/yyy",
										},
										"iamUser": {
											"arn:aws:iam::111222333:user/ecr-login",
										},
									},
									EgressPorts: "1024-65535",
									NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
								},
								AllowedOnboardingTeams: nil,
								Capabilities:           []string{"vpc-peering", "gpu-compute"},
								PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
									{
										ID:      "123r",
										Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
										OwnerID: "ownerxxx",
									},
								},
								LastUpdated: "2021-12-13T05:50:07.492Z",
								Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
							},
						},
						{
							Spec: registryv1.ClusterSpec{
								Name:      "cluster02-prod-euwest1",
								ShortName: "cluster02prodeuwest1",
								APIServer: registryv1.APIServer{
									Endpoint:                 "https://cluster02-prod-euwest1.example.com",
									CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0==",
								},
								ArgoInstance:           "argocd-prod-gen-02.cluster02-prod-useast1.example.com",
								Region:                 "euwest1",
								CloudType:              "azure",
								Environment:            "Prod",
								BusinessUnit:           "BU2",
								ChargebackBusinessUnit: "BU2",
								ChargedBack:            ptr.To(false),
								Offering:               []registryv1.Offering{"caas", "paas"},
								AccountID:              "11111-2222-3333-4444-55555555",
								Tiers: []registryv1.Tier{
									{
										Name:              "worker",
										InstanceType:      "c5.9xlarge",
										MinCapacity:       3,
										MaxCapacity:       1000,
										EnableKataSupport: false,
									},
									{
										Name:         "workerMemoryOptimized",
										InstanceType: "r5.8xlarge",
										MinCapacity:  0,
										MaxCapacity:  100,
										Labels: map[string]string{
											"node.kubernetes.io/workload.memory-optimized": "true",
										},
										Taints: []string{
											"workload=memory-optimized:NoSchedule",
										},
										EnableKataSupport: false,
										KernelParameters:  nil,
									},
								},
								VirtualNetworks: []registryv1.VirtualNetwork{
									{
										ID:    "/subscriptions/11111-2222-3333-4444-55555555/resourceGroups/cluster02_prod_euwest1_network/providers/Microsoft.Network/virtualNetworks/cluster02_prod_euwest1_network-vnet/subnets/cluster02_prod_euwest1_network_10_3_0_0_24",
										Cidrs: []string{"10.3.0.0/24"},
									},
								},
								RegisteredAt:     "2019-02-10T06:15:32Z",
								Status:           "Active",
								Phase:            "Upgrading",
								Type:             "Dedicated",
								MaintenanceGroup: "B",
								Extra: registryv1.Extra{
									DomainName: "example.com",
									LbEndpoints: map[string]string{
										"public": "cluster02-prod-euwest1.example.com",
									},
									LoggingEndpoints: nil,
									EcrIamArns: map[string][]string{
										"iamRoles": {
											"arn:aws:iam::account-id:role/xxx",
										},
										"iamUser": {
											"arn:aws:iam::461989703686:user/ecr-login",
										},
									},
								},
								Capabilities: []string{"mct-support"},
								LastUpdated:  "2020-02-10T06:15:32Z",
								Tags:         map[string]string{"onboarding": "off", "scaling": "on"},
							},
						},
						{
							Spec: registryv1.ClusterSpec{
								Name:      "cluster03-prod-uswest1",
								ShortName: "cluster03produswest1",
								APIServer: registryv1.APIServer{
									Endpoint:                 "https://cluster03-prod-uswest1.example.com",
									CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS==",
								},
								ArgoInstance:           "argocd-prod-gen-03.cluster03-prod-useast1.example.com",
								Region:                 "uswest1",
								CloudType:              "aws",
								Environment:            "Prod",
								BusinessUnit:           "BU1",
								ChargebackBusinessUnit: "BU1",
								ChargedBack:            ptr.To(true),
								Offering:               []registryv1.Offering{"paas"},
								AccountID:              "12345678",
								Tiers: []registryv1.Tier{
									{
										Name:              "proxy",
										InstanceType:      "r5a.4xlarge",
										MinCapacity:       3,
										MaxCapacity:       200,
										EnableKataSupport: false,
									},
									{
										Name:         "worker",
										InstanceType: "c5.9xlarge",
										MinCapacity:  3,
										MaxCapacity:  1000,
										Labels: map[string]string{
											"node.kubernetes.io/workload.memory-optimized": "true",
										},
										EnableKataSupport: false,
									},
								},
								VirtualNetworks: []registryv1.VirtualNetwork{
									{
										ID:    "vpc-123456",
										Cidrs: []string{"10.0.22.0/8"},
									},
								},
								RegisteredAt:     "2020-03-19T07:55:46.132Z",
								Status:           "Active",
								Phase:            "Running",
								Type:             "Dedicated",
								MaintenanceGroup: "B",
								Capabilities:     []string{"gpu-compute"},
								AvailabilityZones: []registryv1.AvailabilityZone{
									{
										Name: "us-east-1a",
										ID:   "use1-az1",
									},
									{
										Name: "us-east-1b",
										ID:   "use1-az2",
									},
									{
										Name: "us-east-1c",
										ID:   "use1-az3",
									},
								},
								LastUpdated: "2020-03-20T07:55:46.132Z",
								Tags:        map[string]string{"onboarding": "on", "scaling": "on"},
							},
						},
					},
				},
				{
					name: "invalid lastUpdate parameter format",
					queryParams: map[string]string{
						"region":      "",
						"environment": "",
						"status":      "",
						"lastUpdated": "2020-03-19",
					},
					offset:           0,
					limit:            10,
					expectedCount:    1,
					expectedMore:     false,
					expectedError:    fmt.Errorf("Error converting lastUpdated parameter to RFC3339"),
					expectedClusters: nil,
				},
				{
					name: "valid lastUpdate parameter format",
					queryParams: map[string]string{
						"region":      "",
						"environment": "",
						"status":      "",
						"lastUpdated": "2021-12-13T00:50:07.492Z",
					},
					offset:        0,
					limit:         10,
					expectedCount: 1,
					expectedMore:  false,
					expectedError: nil,
					expectedClusters: []registryv1.Cluster{
						{
							Spec: registryv1.ClusterSpec{
								Name:      "cluster01-prod-useast1",
								ShortName: "cluster01produseast1",
								APIServer: registryv1.APIServer{
									Endpoint:                 "https://cluster01-prod-useast1.example.com",
									CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
								},
								ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
								Region:                 "useast1",
								CloudType:              "azure",
								Environment:            "Prod",
								BusinessUnit:           "BU1",
								ChargebackBusinessUnit: "BU1",
								ChargedBack:            ptr.To(true),
								Offering:               []registryv1.Offering{"caas", "paas"},
								AccountID:              "11111-2222-3333-4444-555555555",
								Tiers: []registryv1.Tier{
									{
										Name:              "worker",
										InstanceType:      "c5.9xlarge",
										MinCapacity:       3,
										MaxCapacity:       1000,
										EnableKataSupport: false,
									},
									{
										Name:         "workerMemoryOptimized",
										InstanceType: "r5.8xlarge",
										MinCapacity:  0,
										MaxCapacity:  100,
										Labels: map[string]string{
											"node.kubernetes.io/workload.memory-optimized": "true",
										},
										Taints:            []string{"workload=memory-optimized:NoSchedule"},
										EnableKataSupport: false,
									},
								},
								VirtualNetworks: []registryv1.VirtualNetwork{
									{
										ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
										Cidrs: []string{"10.0.0.0/24"},
									},
									{
										ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
										Cidrs: []string{"10.1.0.0/24"},
									},
								},
								RegisteredAt:     "2021-12-13T05:50:07.492Z",
								Status:           "Active",
								Phase:            "Running",
								Type:             "Shared",
								MaintenanceGroup: "B",
								Extra: registryv1.Extra{
									DomainName: "example.com",
									LbEndpoints: map[string]string{
										"public": "cluster01-prod-useast1.example.com",
									},
									LoggingEndpoints: []map[string]string{
										{
											"region":    "useast1",
											"endpoint":  "splunk-us-east1.example.com",
											"isDefault": "true",
										},
										{
											"isDefault": "false",
											"region":    "useast2",
											"endpoint":  "splunk-us-east2.example.com",
										},
									},
									EcrIamArns: map[string][]string{
										"iamRoles": {
											"arn:aws:iam::account-id:role/xxx",
											"arn:aws:iam::account-id:role/yyy",
										},
										"iamUser": {
											"arn:aws:iam::111222333:user/ecr-login",
										},
									},
									EgressPorts: "1024-65535",
									NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
								},
								AllowedOnboardingTeams: nil,
								Capabilities:           []string{"vpc-peering", "gpu-compute"},
								PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
									{
										ID:      "123r",
										Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
										OwnerID: "ownerxxx",
									},
								},
								LastUpdated: "2021-12-13T05:50:07.492Z",
								Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
							},
						},
					},
				},
				{
					name: "set all query parameters",
					queryParams: map[string]string{
						"region":      "euwest1",
						"environment": "Prod",
						"status":      "Active",
						"lastUpdated": "2019-12-13T00:50:07.492Z",
					},
					offset:        0,
					limit:         10,
					expectedCount: 1,
					expectedMore:  false,
					expectedError: nil,
					expectedClusters: []registryv1.Cluster{
						{
							Spec: registryv1.ClusterSpec{
								Name:      "cluster02-prod-euwest1",
								ShortName: "cluster02prodeuwest1",
								APIServer: registryv1.APIServer{
									Endpoint:                 "https://cluster02-prod-euwest1.example.com",
									CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0==",
								},
								ArgoInstance:           "argocd-prod-gen-02.cluster02-prod-useast1.example.com",
								Region:                 "euwest1",
								CloudType:              "azure",
								Environment:            "Prod",
								BusinessUnit:           "BU2",
								ChargebackBusinessUnit: "BU2",
								ChargedBack:            ptr.To(false),
								Offering:               []registryv1.Offering{"caas", "paas"},
								AccountID:              "11111-2222-3333-4444-55555555",
								Tiers: []registryv1.Tier{
									{
										Name:              "worker",
										InstanceType:      "c5.9xlarge",
										MinCapacity:       3,
										MaxCapacity:       1000,
										EnableKataSupport: false,
									},
									{
										Name:         "workerMemoryOptimized",
										InstanceType: "r5.8xlarge",
										MinCapacity:  0,
										MaxCapacity:  100,
										Labels: map[string]string{
											"node.kubernetes.io/workload.memory-optimized": "true",
										},
										Taints: []string{
											"workload=memory-optimized:NoSchedule",
										},
										EnableKataSupport: false,
										KernelParameters:  nil,
									},
								},
								VirtualNetworks: []registryv1.VirtualNetwork{
									{
										ID:    "/subscriptions/11111-2222-3333-4444-55555555/resourceGroups/cluster02_prod_euwest1_network/providers/Microsoft.Network/virtualNetworks/cluster02_prod_euwest1_network-vnet/subnets/cluster02_prod_euwest1_network_10_3_0_0_24",
										Cidrs: []string{"10.3.0.0/24"},
									},
								},
								RegisteredAt:     "2019-02-10T06:15:32Z",
								Status:           "Active",
								Phase:            "Upgrading",
								Type:             "Dedicated",
								MaintenanceGroup: "B",
								Extra: registryv1.Extra{
									DomainName: "example.com",
									LbEndpoints: map[string]string{
										"public": "cluster02-prod-euwest1.example.com",
									},
									LoggingEndpoints: nil,
									EcrIamArns: map[string][]string{
										"iamRoles": {
											"arn:aws:iam::account-id:role/xxx",
										},
										"iamUser": {
											"arn:aws:iam::461989703686:user/ecr-login",
										},
									},
								},
								Capabilities: []string{"mct-support"},
								LastUpdated:  "2020-02-10T06:15:32Z",
								Tags:         map[string]string{"onboarding": "off", "scaling": "on"},
							},
						},
					},
				},
			}
			for _, tc := range tcs {

				By(fmt.Sprintf("\tTest %s: When getting all clusters with region:%s, environment:%s, status:%s, lastUpdated:%s, offset:%d, limit:%d",
					tc.name,
					tc.queryParams["region"],
					tc.queryParams["environment"],
					tc.queryParams["status"],
					tc.queryParams["lastUpdated"],
					tc.offset,
					tc.limit))

				clusters, count, more, err := db.ListClusters(
					tc.offset,
					tc.limit,
					tc.queryParams["region"],
					tc.queryParams["environment"],
					tc.queryParams["status"],
					tc.queryParams["lastUpdated"],
				)

				if tc.expectedError != nil {
					Expect(err.Error()).To(ContainSubstring(tc.expectedError.Error()))
					continue
				}

				Expect(err).To(BeNil())

				sort.Slice(clusters, func(i, j int) bool {
					return clusters[i].Spec.Name < clusters[j].Spec.Name
				})

				Expect(count).To(Equal(tc.expectedCount))
				Expect(more).To(Equal(tc.expectedMore))

				for i := 0; i < tc.expectedCount; i++ {
					Expect(clusters[i].Spec).To(Equal(tc.expectedClusters[i].Spec))
				}
			}
		})
	})

	It("Should handle DB List clusters with filter", func() {
		tcs := []struct {
			name             string
			offset           int
			limit            int
			filter           *DynamoDBFilter
			expectedCount    int
			expectedMore     bool
			expectedError    error
			expectedClusters []registryv1.Cluster
		}{
			{
				name:          "empty filter",
				offset:        0,
				limit:         10,
				filter:        NewDynamoDBFilter(),
				expectedCount: 3,
				expectedMore:  false,
				expectedError: nil,
				expectedClusters: []registryv1.Cluster{
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster01-prod-useast1",
							ShortName: "cluster01produseast1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster01-prod-useast1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
							},
							ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
							Region:                 "useast1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-555555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints:            []string{"workload=memory-optimized:NoSchedule"},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
									Cidrs: []string{"10.0.0.0/24"},
								},
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
									Cidrs: []string{"10.1.0.0/24"},
								},
							},
							RegisteredAt:     "2021-12-13T05:50:07.492Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Shared",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster01-prod-useast1.example.com",
								},
								LoggingEndpoints: []map[string]string{
									{
										"region":    "useast1",
										"endpoint":  "splunk-us-east1.example.com",
										"isDefault": "true",
									},
									{
										"isDefault": "false",
										"region":    "useast2",
										"endpoint":  "splunk-us-east2.example.com",
									},
								},
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
										"arn:aws:iam::account-id:role/yyy",
									},
									"iamUser": {
										"arn:aws:iam::111222333:user/ecr-login",
									},
								},
								EgressPorts: "1024-65535",
								NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
							},
							AllowedOnboardingTeams: nil,
							Capabilities:           []string{"vpc-peering", "gpu-compute"},
							PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
								{
									ID:      "123r",
									Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
									OwnerID: "ownerxxx",
								},
							},
							LastUpdated: "2021-12-13T05:50:07.492Z",
							Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
						},
					},
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster02-prod-euwest1",
							ShortName: "cluster02prodeuwest1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster02-prod-euwest1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0==",
							},
							ArgoInstance:           "argocd-prod-gen-02.cluster02-prod-useast1.example.com",
							Region:                 "euwest1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU2",
							ChargebackBusinessUnit: "BU2",
							ChargedBack:            ptr.To(false),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-55555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
									KernelParameters:  nil,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-55555555/resourceGroups/cluster02_prod_euwest1_network/providers/Microsoft.Network/virtualNetworks/cluster02_prod_euwest1_network-vnet/subnets/cluster02_prod_euwest1_network_10_3_0_0_24",
									Cidrs: []string{"10.3.0.0/24"},
								},
							},
							RegisteredAt:     "2019-02-10T06:15:32Z",
							Status:           "Active",
							Phase:            "Upgrading",
							Type:             "Dedicated",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster02-prod-euwest1.example.com",
								},
								LoggingEndpoints: nil,
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
									},
									"iamUser": {
										"arn:aws:iam::461989703686:user/ecr-login",
									},
								},
							},
							Capabilities: []string{"mct-support"},
							LastUpdated:  "2020-02-10T06:15:32Z",
							Tags:         map[string]string{"onboarding": "off", "scaling": "on"},
						},
					},
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster03-prod-uswest1",
							ShortName: "cluster03produswest1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster03-prod-uswest1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS==",
							},
							ArgoInstance:           "argocd-prod-gen-03.cluster03-prod-useast1.example.com",
							Region:                 "uswest1",
							CloudType:              "aws",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"paas"},
							AccountID:              "12345678",
							Tiers: []registryv1.Tier{
								{
									Name:              "proxy",
									InstanceType:      "r5a.4xlarge",
									MinCapacity:       3,
									MaxCapacity:       200,
									EnableKataSupport: false,
								},
								{
									Name:         "worker",
									InstanceType: "c5.9xlarge",
									MinCapacity:  3,
									MaxCapacity:  1000,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "vpc-123456",
									Cidrs: []string{"10.0.22.0/8"},
								},
							},
							RegisteredAt:     "2020-03-19T07:55:46.132Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Dedicated",
							MaintenanceGroup: "B",
							Capabilities:     []string{"gpu-compute"},
							LastUpdated:      "2020-03-20T07:55:46.132Z",
							Tags:             map[string]string{"onboarding": "on", "scaling": "on"},
							AvailabilityZones: []registryv1.AvailabilityZone{
								{
									Name: "us-east-1a",
									ID:   "use1-az1",
								},
								{
									Name: "us-east-1b",
									ID:   "use1-az2",
								},
								{
									Name: "us-east-1c",
									ID:   "use1-az3",
								},
							},
						},
					},
				},
			},
			{
				name:          "single valid condition with equals operand",
				offset:        0,
				limit:         10,
				filter:        NewDynamoDBFilter().AddCondition(models.NewFilterCondition("crd.spec.cloudType", "=", "azure")),
				expectedCount: 2,
				expectedMore:  false,
				expectedError: nil,
				expectedClusters: []registryv1.Cluster{
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster01-prod-useast1",
							ShortName: "cluster01produseast1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster01-prod-useast1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
							},
							ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
							Region:                 "useast1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-555555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints:            []string{"workload=memory-optimized:NoSchedule"},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
									Cidrs: []string{"10.0.0.0/24"},
								},
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
									Cidrs: []string{"10.1.0.0/24"},
								},
							},
							RegisteredAt:     "2021-12-13T05:50:07.492Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Shared",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster01-prod-useast1.example.com",
								},
								LoggingEndpoints: []map[string]string{
									{
										"region":    "useast1",
										"endpoint":  "splunk-us-east1.example.com",
										"isDefault": "true",
									},
									{
										"isDefault": "false",
										"region":    "useast2",
										"endpoint":  "splunk-us-east2.example.com",
									},
								},
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
										"arn:aws:iam::account-id:role/yyy",
									},
									"iamUser": {
										"arn:aws:iam::111222333:user/ecr-login",
									},
								},
								EgressPorts: "1024-65535",
								NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
							},
							AllowedOnboardingTeams: nil,
							Capabilities:           []string{"vpc-peering", "gpu-compute"},
							PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
								{
									ID:      "123r",
									Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
									OwnerID: "ownerxxx",
								},
							},
							LastUpdated: "2021-12-13T05:50:07.492Z",
							Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
						},
					},
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster02-prod-euwest1",
							ShortName: "cluster02prodeuwest1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster02-prod-euwest1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0==",
							},
							ArgoInstance:           "argocd-prod-gen-02.cluster02-prod-useast1.example.com",
							Region:                 "euwest1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU2",
							ChargebackBusinessUnit: "BU2",
							ChargedBack:            ptr.To(false),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-55555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
									KernelParameters:  nil,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-55555555/resourceGroups/cluster02_prod_euwest1_network/providers/Microsoft.Network/virtualNetworks/cluster02_prod_euwest1_network-vnet/subnets/cluster02_prod_euwest1_network_10_3_0_0_24",
									Cidrs: []string{"10.3.0.0/24"},
								},
							},
							RegisteredAt:     "2019-02-10T06:15:32Z",
							Status:           "Active",
							Phase:            "Upgrading",
							Type:             "Dedicated",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster02-prod-euwest1.example.com",
								},
								LoggingEndpoints: nil,
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
									},
									"iamUser": {
										"arn:aws:iam::461989703686:user/ecr-login",
									},
								},
							},
							Capabilities: []string{"mct-support"},
							LastUpdated:  "2020-02-10T06:15:32Z",
							Tags:         map[string]string{"onboarding": "off", "scaling": "on"},
						},
					},
				},
			},
			{
				name:          "single valid condition with date comparison",
				offset:        0,
				limit:         10,
				filter:        NewDynamoDBFilter().AddCondition(models.NewFilterCondition("crd.spec.lastUpdated", ">", "2020-02-20T00:00:00Z")),
				expectedCount: 2,
				expectedMore:  false,
				expectedError: nil,
				expectedClusters: []registryv1.Cluster{
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster01-prod-useast1",
							ShortName: "cluster01produseast1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster01-prod-useast1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
							},
							ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
							Region:                 "useast1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-555555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints:            []string{"workload=memory-optimized:NoSchedule"},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
									Cidrs: []string{"10.0.0.0/24"},
								},
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
									Cidrs: []string{"10.1.0.0/24"},
								},
							},
							RegisteredAt:     "2021-12-13T05:50:07.492Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Shared",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster01-prod-useast1.example.com",
								},
								LoggingEndpoints: []map[string]string{
									{
										"region":    "useast1",
										"endpoint":  "splunk-us-east1.example.com",
										"isDefault": "true",
									},
									{
										"isDefault": "false",
										"region":    "useast2",
										"endpoint":  "splunk-us-east2.example.com",
									},
								},
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
										"arn:aws:iam::account-id:role/yyy",
									},
									"iamUser": {
										"arn:aws:iam::111222333:user/ecr-login",
									},
								},
								EgressPorts: "1024-65535",
								NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
							},
							AllowedOnboardingTeams: nil,
							Capabilities:           []string{"vpc-peering", "gpu-compute"},
							PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
								{
									ID:      "123r",
									Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
									OwnerID: "ownerxxx",
								},
							},
							LastUpdated: "2021-12-13T05:50:07.492Z",
							Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
						},
					},
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster03-prod-uswest1",
							ShortName: "cluster03produswest1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster03-prod-uswest1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS==",
							},
							ArgoInstance:           "argocd-prod-gen-03.cluster03-prod-useast1.example.com",
							Region:                 "uswest1",
							CloudType:              "aws",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"paas"},
							AccountID:              "12345678",
							Tiers: []registryv1.Tier{
								{
									Name:              "proxy",
									InstanceType:      "r5a.4xlarge",
									MinCapacity:       3,
									MaxCapacity:       200,
									EnableKataSupport: false,
								},
								{
									Name:         "worker",
									InstanceType: "c5.9xlarge",
									MinCapacity:  3,
									MaxCapacity:  1000,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "vpc-123456",
									Cidrs: []string{"10.0.22.0/8"},
								},
							},
							RegisteredAt:     "2020-03-19T07:55:46.132Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Dedicated",
							MaintenanceGroup: "B",
							Capabilities:     []string{"gpu-compute"},
							LastUpdated:      "2020-03-20T07:55:46.132Z",
							Tags:             map[string]string{"onboarding": "on", "scaling": "on"},
							AvailabilityZones: []registryv1.AvailabilityZone{
								{
									Name: "us-east-1a",
									ID:   "use1-az1",
								},
								{
									Name: "us-east-1b",
									ID:   "use1-az2",
								},
								{
									Name: "us-east-1c",
									ID:   "use1-az3",
								},
							},
						},
					},
				},
			},
			{
				name:   "multiple valid conditions",
				offset: 0,
				limit:  10,
				filter: NewDynamoDBFilter().
					AddCondition(models.NewFilterCondition("crd.spec.cloudType", "=", "azure")).
					AddCondition(models.NewFilterCondition("crd.spec.environment", "=", "Prod")).
					AddCondition(models.NewFilterCondition("crd.spec.status", "=", "Active")),
				expectedCount: 2,
				expectedMore:  false,
				expectedError: nil,
				expectedClusters: []registryv1.Cluster{
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster01-prod-useast1",
							ShortName: "cluster01produseast1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster01-prod-useast1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJ==",
							},
							ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
							Region:                 "useast1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU1",
							ChargebackBusinessUnit: "BU1",
							ChargedBack:            ptr.To(true),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-555555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints:            []string{"workload=memory-optimized:NoSchedule"},
									EnableKataSupport: false,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_master_network_10_0_0_0_24",
									Cidrs: []string{"10.0.0.0/24"},
								},
								{
									ID:    "/subscriptions/11111-2222-3333-4444-555555555/resourceGroups/cluster01_prod_useast1_network/providers/Microsoft.Network/virtualNetworks/cluster01_prod_useast1-vnet/subnets/cluster01_prod_useast1_worker_network_10_1_0_0_24",
									Cidrs: []string{"10.1.0.0/24"},
								},
							},
							RegisteredAt:     "2021-12-13T05:50:07.492Z",
							Status:           "Active",
							Phase:            "Running",
							Type:             "Shared",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster01-prod-useast1.example.com",
								},
								LoggingEndpoints: []map[string]string{
									{
										"region":    "useast1",
										"endpoint":  "splunk-us-east1.example.com",
										"isDefault": "true",
									},
									{
										"isDefault": "false",
										"region":    "useast2",
										"endpoint":  "splunk-us-east2.example.com",
									},
								},
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
										"arn:aws:iam::account-id:role/yyy",
									},
									"iamUser": {
										"arn:aws:iam::111222333:user/ecr-login",
									},
								},
								EgressPorts: "1024-65535",
								NFSInfo:     []map[string]string{{"endpoint": "xyz", "basePath": "xyz", "name": "xxxss5"}},
							},
							AllowedOnboardingTeams: nil,
							Capabilities:           []string{"vpc-peering", "gpu-compute"},
							PeerVirtualNetworks: []registryv1.PeerVirtualNetwork{
								{
									ID:      "123r",
									Cidrs:   []string{"10.2.0.1/23", "10.3.0.1/24"},
									OwnerID: "ownerxxx",
								},
							},
							LastUpdated: "2021-12-13T05:50:07.492Z",
							Tags:        map[string]string{"onboarding": "off", "scaling": "off"},
						},
					},
					{
						Spec: registryv1.ClusterSpec{
							Name:      "cluster02-prod-euwest1",
							ShortName: "cluster02prodeuwest1",
							APIServer: registryv1.APIServer{
								Endpoint:                 "https://cluster02-prod-euwest1.example.com",
								CertificateAuthorityData: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0==",
							},
							ArgoInstance:           "argocd-prod-gen-02.cluster02-prod-useast1.example.com",
							Region:                 "euwest1",
							CloudType:              "azure",
							Environment:            "Prod",
							BusinessUnit:           "BU2",
							ChargebackBusinessUnit: "BU2",
							ChargedBack:            ptr.To(false),
							Offering:               []registryv1.Offering{"caas", "paas"},
							AccountID:              "11111-2222-3333-4444-55555555",
							Tiers: []registryv1.Tier{
								{
									Name:              "worker",
									InstanceType:      "c5.9xlarge",
									MinCapacity:       3,
									MaxCapacity:       1000,
									EnableKataSupport: false,
								},
								{
									Name:         "workerMemoryOptimized",
									InstanceType: "r5.8xlarge",
									MinCapacity:  0,
									MaxCapacity:  100,
									Labels: map[string]string{
										"node.kubernetes.io/workload.memory-optimized": "true",
									},
									Taints: []string{
										"workload=memory-optimized:NoSchedule",
									},
									EnableKataSupport: false,
									KernelParameters:  nil,
								},
							},
							VirtualNetworks: []registryv1.VirtualNetwork{
								{
									ID:    "/subscriptions/11111-2222-3333-4444-55555555/resourceGroups/cluster02_prod_euwest1_network/providers/Microsoft.Network/virtualNetworks/cluster02_prod_euwest1_network-vnet/subnets/cluster02_prod_euwest1_network_10_3_0_0_24",
									Cidrs: []string{"10.3.0.0/24"},
								},
							},
							RegisteredAt:     "2019-02-10T06:15:32Z",
							Status:           "Active",
							Phase:            "Upgrading",
							Type:             "Dedicated",
							MaintenanceGroup: "B",
							Extra: registryv1.Extra{
								DomainName: "example.com",
								LbEndpoints: map[string]string{
									"public": "cluster02-prod-euwest1.example.com",
								},
								LoggingEndpoints: nil,
								EcrIamArns: map[string][]string{
									"iamRoles": {
										"arn:aws:iam::account-id:role/xxx",
									},
									"iamUser": {
										"arn:aws:iam::461989703686:user/ecr-login",
									},
								},
							},
							Capabilities: []string{"mct-support"},
							LastUpdated:  "2020-02-10T06:15:32Z",
							Tags:         map[string]string{"onboarding": "off", "scaling": "on"},
						},
					},
				},
			},
		}
		for _, tc := range tcs {

			By(fmt.Sprintf("\tTest %s: When getting clusters by filter:%s, offset:%d, limit:%d",
				tc.name,
				tc.filter,
				tc.offset,
				tc.limit))

			clusters, count, more, err := db.ListClustersWithFilter(
				tc.offset,
				tc.limit,
				tc.filter,
			)

			if tc.expectedError != nil {
				Expect(err.Error()).To(ContainSubstring(tc.expectedError.Error()))
				continue
			}

			Expect(err).To(BeNil())

			sort.Slice(clusters, func(i, j int) bool {
				return clusters[i].Spec.Name < clusters[j].Spec.Name
			})

			Expect(count).To(Equal(tc.expectedCount))
			Expect(more).To(Equal(tc.expectedMore))

			for i := 0; i < tc.expectedCount; i++ {
				Expect(clusters[i].Spec).To(Equal(tc.expectedClusters[i].Spec))
			}
		}
	})
})
