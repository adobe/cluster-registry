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
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/assert"
)

type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
	clusters map[string]*ClusterDb
}

func (m *mockDynamoDBClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {

	resp, err := dynamodbattribute.MarshalMap(
		m.clusters[*input.Key["name"].S],
	)
	if err != nil {
		return nil, err
	}

	output := &dynamodb.GetItemOutput{
		Item: resp,
	}
	return output, nil
}

func (m *mockDynamoDBClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {

	clusterName := *input.Key["name"].S

	if c, exist := m.clusters[clusterName]; exist {
		delete(m.clusters, c.Name)
	} else {
		return nil, errors.New("cluster not found")
	}

	output := &dynamodb.DeleteItemOutput{}
	return output, nil
}

func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	var cluster ClusterDb
	err := dynamodbattribute.UnmarshalMap(input.Item, &cluster)
	if err != nil {
		return nil, err
	}

	m.clusters[cluster.Name] = &cluster
	return &dynamodb.PutItemOutput{}, err
}

func (m *mockDynamoDBClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	var resp []map[string]*dynamodb.AttributeValue

	for _, c := range m.clusters {
		attr, err := dynamodbattribute.MarshalMap(
			c,
		)
		if err != nil {
			return nil, err
		}
		resp = append(resp, attr)
	}

	output := &dynamodb.ScanOutput{
		Items: resp,
	}
	return output, nil
}

func TestNewDb(t *testing.T) {
	test := assert.New(t)

	os.Setenv("DB_ENDPOINT", "dummy-url")
	os.Setenv("DB_AWS_REGION", "dummy-region")

	m := monitoring.NewMetrics("cluster_registry_api_database_test", nil, true)
	d := NewDb(m)
	test.NotNil(d)
}

func TestPutCluster(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name             string
		dbClusters       map[string]*ClusterDb
		newCluster       *registryv1.Cluster
		expectedClusters []registryv1.Cluster
		expectedError    error
	}{
		{
			name:       "new cluster",
			dbClusters: map[string]*ClusterDb{},
			newCluster: &registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster2",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster2",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			},
			expectedError: nil,
		},
		{
			name: "existing cluster",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					Name: "cluster1",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster1",
							LastUpdated:  "2020-02-13T06:15:32Z",
							RegisteredAt: "2019-02-13T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				}},
			newCluster: &registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			},
			expectedError: nil,
		},
	}

	for _, tc := range tcs {
		db := &db{
			dbAPI:     &mockDynamoDBClient{clusters: tc.dbClusters},
			tableName: "cluster-registry",
			met:       monitoring.NewMetrics("cluster_registry_api_database_test", nil, true),
		}

		err := db.PutCluster(tc.newCluster)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}

		clusters, _, err := db.ListClusters("", "", "", "")
		test.NoError(err)

		test.Equal(tc.expectedClusters, clusters)
	}
}

func TestDeleteCluster(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name             string
		clusterName      string
		dbClusters       map[string]*ClusterDb
		expectedClusters []registryv1.Cluster
		expectedError    error
	}{
		{
			name:        "existing cluster",
			clusterName: "cluster1",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					Name: "cluster1",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster1",
							LastUpdated:  "2020-02-13T06:15:32Z",
							RegisteredAt: "2019-02-13T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				}},
			expectedClusters: []registryv1.Cluster{},
			expectedError:    nil,
		},
		{
			name:        "non existing cluster",
			clusterName: "cluster2",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					Name: "cluster1",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster1",
							LastUpdated:  "2020-02-13T06:15:32Z",
							RegisteredAt: "2019-02-13T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				}},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}},
			},
			expectedError: fmt.Errorf("cluster not found"),
		},
	}

	for _, tc := range tcs {
		db := &db{
			dbAPI:     &mockDynamoDBClient{clusters: tc.dbClusters},
			tableName: "cluster-registry",
			met:       monitoring.NewMetrics("cluster_registry_api_database_test", nil, true),
		}

		err := db.DeleteCluster(tc.clusterName)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}

		c, _, err := db.ListClusters("", "", "", "")
		test.NoError(err)

		test.Equal(tc.expectedClusters, c)
	}
}

func TestGetCluster(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name            string
		clusterName     string
		dbClusters      map[string]*ClusterDb
		expectedCluster *registryv1.Cluster
		expectedError   error
	}{
		{
			name:        "existing cluster",
			clusterName: "cluster1",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					Name: "cluster1",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster1",
							LastUpdated:  "2020-02-13T06:15:32Z",
							RegisteredAt: "2019-02-13T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				}},
			expectedCluster: &registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}},
			expectedError: nil,
		},
		{
			name:        "non existing cluster",
			clusterName: "cluster2",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					Name: "cluster1",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster1",
							LastUpdated:  "2020-02-13T06:15:32Z",
							RegisteredAt: "2019-02-13T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				}},
			expectedCluster: nil,
			expectedError:   nil,
		},
	}

	for _, tc := range tcs {
		db := &db{
			dbAPI:     &mockDynamoDBClient{clusters: tc.dbClusters},
			tableName: "cluster-registry",
			met:       monitoring.NewMetrics("cluster_registry_api_database_test", nil, true),
		}

		c, err := db.GetCluster(tc.clusterName)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}
		test.Equal(tc.expectedCluster, c)
	}
}

// TODO: add tests for filtering
func TestListClusters(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name             string
		queryParams      map[string]string
		dbClusters       map[string]*ClusterDb
		expectedClusters []registryv1.Cluster
		expectedError    error
	}{
		{
			name: "all clusters",
			queryParams: map[string]string{
				"region":       "",
				"environment":  "",
				"businessUnit": "",
				"status":       "",
			},
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					Name: "cluster1",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster1",
							LastUpdated:  "2020-02-13T06:15:32Z",
							RegisteredAt: "2019-02-13T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
						},
					},
				},
				"cluster2": {
					Name: "cluster2",
					Cluster: &registryv1.Cluster{
						Spec: registryv1.ClusterSpec{
							Name:         "cluster2",
							LastUpdated:  "2020-02-14T06:15:32Z",
							RegisteredAt: "2019-02-14T06:15:32Z",
							Status:       "Active",
							Phase:        "Running",
							Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
						},
					},
				}},
			expectedClusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}}, {
				Spec: registryv1.ClusterSpec{
					Name:         "cluster2",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			},
			expectedError: nil,
		},
	}

	for _, tc := range tcs {
		db := &db{
			dbAPI:     &mockDynamoDBClient{clusters: tc.dbClusters},
			tableName: "cluster-registry",
			met:       monitoring.NewMetrics("cluster_registry_api_database_test", nil, true),
		}

		clusters, _, err := db.ListClusters(
			tc.queryParams["region"], tc.queryParams["environments"],
			tc.queryParams["businessUnit"], tc.queryParams["status"])

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}

		sort.Slice(clusters, func(i, j int) bool {
			return clusters[i].Spec.Name < clusters[j].Spec.Name
		})

		test.Equal(tc.expectedClusters, clusters)
	}
}
