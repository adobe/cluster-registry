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

package database

import (
	"fmt"
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
							Status:       "Deprecated",
							Phase:        "Running",
							Type:         "Shared",
							Capabilities: []string{"vpc-peering", "gpu-compute"},
							Tags:         map[string]string{"onboarding": "off", "scaling": "off"},
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
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				},
				{
					name:        "update existing cluster",
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
				expectedClusters []registryv1.Cluster
			}{
				{
					name: "all clusters",
					queryParams: map[string]string{
						"region":      "",
						"environment": "",
						"status":      "",
					},
					offset:        0,
					limit:         10,
					expectedCount: 3,
					expectedMore:  false,
					expectedClusters: []registryv1.Cluster{
						{
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
								Status:       "Deprecated",
								Phase:        "Running",
								Type:         "Shared",
								Capabilities: []string{"vpc-peering", "gpu-compute"},
								Tags:         map[string]string{"onboarding": "off", "scaling": "off"},
							},
						},
						{
							Spec: registryv1.ClusterSpec{
								Name:        "cluster02-prod-euwest1",
								Region:      "euwest1",
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
								Phase:        "Upgrading",
								Type:         "Dedicated",
								Capabilities: []string{"mct-support"},
								Tags:         map[string]string{"onboarding": "off", "scaling": "on"},
							},
						},
						{
							Spec: registryv1.ClusterSpec{
								Name:        "cluster03-prod-uswest1",
								Region:      "uswest1",
								Environment: "Prod",
								Offering:    []registryv1.Offering{"paas"},
								Tiers: []registryv1.Tier{
									{
										Name:              "proxy",
										InstanceType:      "",
										ContainerRuntime:  "",
										MinCapacity:       0,
										MaxCapacity:       0,
										EnableKataSupport: false,
									},
									{
										Name:        "worker",
										MinCapacity: 0,
										MaxCapacity: 0,
										Labels: map[string]string{
											"node.kubernetes.io/workload.memory-optimized": "true",
										},
										EnableKataSupport: false,
									},
								},
								Status:       "Active",
								Phase:        "Running",
								Type:         "Dedicated",
								Capabilities: []string{"gpu-compute"},
								Tags:         map[string]string{"scaling": "on", "onboarding": "on"},
							},
						},
					},
				},
			}

			for _, tc := range tcs {

				By(fmt.Sprintf("\tTest %s:\tWhen getting all clusters with region:%s, environment:%s, status:%s, offset:%d, limit:%d",
					tc.name, tc.queryParams["region"], tc.queryParams["environment"], tc.queryParams["status"], tc.offset, tc.limit))

				clusters, count, more, err := db.ListClusters(
					tc.offset,
					tc.limit,
					tc.queryParams["region"],
					tc.queryParams["environment"],
					tc.queryParams["status"])

				Expect(err).To(BeNil())

				sort.Slice(clusters, func(i, j int) bool {
					return clusters[i].Spec.Name < clusters[j].Spec.Name
				})

				Expect(count, tc.expectedCount)
				Expect(more, tc.expectedMore)

				for i := 0; i < tc.expectedCount; i++ {
					Expect(clusters[i].Spec).To(Equal(tc.expectedClusters[i].Spec))
				}
			}
		})
	})
})
