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
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/stretchr/testify/assert"
)

type mockDynamoDBClient struct {
	dynamodbiface.DynamoDBAPI
	clusters map[string]*ClusterDb
}

func (m *mockDynamoDBClient) DescribeTableWithContext(_ context.Context, input *dynamodb.DescribeTableInput, _ ...request.Option) (*dynamodb.DescribeTableOutput, error) {

	if *input.TableName == "mock-clusters" {
		return &dynamodb.DescribeTableOutput{}, nil
	}
	return &dynamodb.DescribeTableOutput{}, errors.New("No sqs found with the name " + *input.TableName)
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
		delete(m.clusters, c.TablePartitionKey)
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

	m.clusters[cluster.TablePartitionKey] = &cluster
	return &dynamodb.PutItemOutput{}, err
}

func (m *mockDynamoDBClient) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
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

	output := &dynamodb.QueryOutput{
		Items: resp,
	}
	return output, nil
}

func TestNewDb(t *testing.T) {
	test := assert.New(t)

	t.Log("Test initializing the database.")

	appConfig := &config.AppConfig{
		DbEndpoint:  "dummy-url",
		DbAwsRegion: "dummy-region",
	}

	m := monitoring.NewMetrics("cluster_registry_api_database_test", true)
	d := NewDb(appConfig, m)
	test.NotNil(d)
}

func TestStatusHealthCheck(t *testing.T) {
	test := assert.New(t)
	t.Log("Test the health check for the database")

	tcs := []struct {
		name          string
		tableName     string
		expectedError error
	}{
		{
			name:          "unhealthy hatabase test",
			tableName:     "mock-clusters",
			expectedError: nil,
		},
		{
			name:          "unhealthy hatabase test",
			tableName:     "missing-mock-clusters",
			expectedError: errors.New("No sqs found with the name missing-mock-clusters"),
		},
	}

	for _, tc := range tcs {

		db := &db{
			dbAPI:   &mockDynamoDBClient{},
			table:   dbTable{name: tc.tableName},
			metrics: monitoring.NewMetrics("cluster_registry_api_database_test", true),
		}

		err := db.Status()
		test.Equal(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError))
	}
}

func TestGetCluster(t *testing.T) {
	test := assert.New(t)

	t.Log("Test getting a single cluster from the database.")

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
					TablePartitionKey: "cluster1",
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
					TablePartitionKey: "cluster1",
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
			expectedError: nil,
		},
	}

	for _, tc := range tcs {

		db := &db{
			dbAPI:   &mockDynamoDBClient{clusters: tc.dbClusters},
			table:   dbTable{name: "cluster-registry-test", partitionKey: "name", searchKey: ""},
			index:   dbTable{name: "cluster-registry-search-test", partitionKey: "kind", searchKey: "name"},
			metrics: monitoring.NewMetrics("cluster_registry_api_database_test", true),
		}

		t.Logf("\tTest %s:\tWhen getting cluster %s", tc.name, tc.clusterName)

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

func TestPutCluster(t *testing.T) {
	test := assert.New(t)

	t.Log("Test create or update a single cluster into the database.")

	tcs := []struct {
		name             string
		dbClusters       map[string]*ClusterDb
		newCluster       *registryv1.Cluster
		offset           int
		limit            int
		expectedClusters []registryv1.Cluster
		expectedCount    int
		expectedMore     bool
		expectedError    error
	}{
		{
			name:       "new cluster",
			dbClusters: map[string]*ClusterDb{},
			offset:     0,
			limit:      10,
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
			expectedCount: 1,
			expectedMore:  false,
			expectedError: nil,
		},
		{
			name: "existing cluster",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					TablePartitionKey: "cluster1",
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
			offset: 0,
			limit:  10,
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
			expectedCount: 1,
			expectedMore:  false,
			expectedError: nil,
		},
	}

	for _, tc := range tcs {

		db := &db{
			dbAPI:   &mockDynamoDBClient{clusters: tc.dbClusters},
			table:   dbTable{name: "cluster-registry-test", partitionKey: "name", searchKey: ""},
			index:   dbTable{name: "cluster-registry-search-test", partitionKey: "kind", searchKey: "name"},
			metrics: monitoring.NewMetrics("cluster_registry_api_database_test", true),
		}

		t.Logf("\tTest %s:\tWhen creating or updating a cluster %s; nr. of items %d", tc.name, tc.newCluster.ClusterName, tc.expectedCount)

		err := db.PutCluster(tc.newCluster)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}

		clusters, count, more, err := db.ListClusters(tc.offset, tc.limit, "", "", "")

		test.NoError(err)
		test.Equal(tc.expectedClusters, clusters)
		test.Equal(tc.expectedCount, count)
		test.Equal(tc.expectedMore, more)
	}
}

func TestDeleteCluster(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name             string
		clusterName      string
		dbClusters       map[string]*ClusterDb
		offset           int
		limit            int
		expectedClusters []registryv1.Cluster
		expectedError    error
		expectedCount    int
		expectedMore     bool
	}{
		{
			name:        "existing cluster",
			clusterName: "cluster1",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					TablePartitionKey: "cluster1",
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
			offset:           0,
			limit:            10,
			expectedClusters: []registryv1.Cluster{},
			expectedCount:    0,
			expectedMore:     false,
			expectedError:    nil,
		},
		{
			name:        "non existing cluster",
			clusterName: "cluster2",
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					TablePartitionKey: "cluster1",
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
			offset: 0,
			limit:  10,
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
			expectedCount: 1,
			expectedMore:  false,
			expectedError: fmt.Errorf("cluster not found"),
		},
	}

	for _, tc := range tcs {

		db := &db{
			dbAPI:   &mockDynamoDBClient{clusters: tc.dbClusters},
			table:   dbTable{name: "cluster-registry-test", partitionKey: "name", searchKey: ""},
			index:   dbTable{name: "cluster-registry-search-test", partitionKey: "kind", searchKey: "name"},
			metrics: monitoring.NewMetrics("cluster_registry_api_database_test", true),
		}

		err := db.DeleteCluster(tc.clusterName)

		if tc.expectedError != nil {
			test.Error(err, "there should be an error processing the message")
			test.Contains(fmt.Sprintf("%v", err), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
		} else {
			test.NoError(err)
		}

		clusters, count, more, err := db.ListClusters(tc.offset, tc.limit, "", "", "")

		test.NoError(err)
		test.Equal(tc.expectedClusters, clusters)
		test.Equal(tc.expectedCount, count)
		test.Equal(tc.expectedMore, more)
	}
}

// TODO: add tests for filtering
func TestListClusters(t *testing.T) {
	test := assert.New(t)

	t.Log("Test getting all clusters from the database.")

	tcs := []struct {
		name             string
		queryParams      map[string]string
		dbClusters       map[string]*ClusterDb
		offset           int
		limit            int
		expectedClusters []registryv1.Cluster
		expectedError    error
		expectedCount    int
		expectedMore     bool
	}{
		{
			name: "all clusters",
			queryParams: map[string]string{
				"region":      "",
				"environment": "",
				"status":      "",
			},
			dbClusters: map[string]*ClusterDb{
				"cluster1": {
					TablePartitionKey: "cluster1",
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
					TablePartitionKey: "cluster2",
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
			offset: 0,
			limit:  200,
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
			expectedCount: 2,
			expectedMore:  false,
			expectedError: nil,
		},
	}

	for _, tc := range tcs {
		db := &db{
			dbAPI:   &mockDynamoDBClient{clusters: tc.dbClusters},
			table:   dbTable{name: "cluster-registry-test", partitionKey: "name", searchKey: ""},
			index:   dbTable{name: "cluster-registry-search-test", partitionKey: "kind", searchKey: "name"},
			metrics: monitoring.NewMetrics("cluster_registry_api_database_test", true),
		}

		t.Logf("\tTest %s:\tWhen getting all clusters with region:%s, environment:%s, status:%s, offset:%d, limit:%d, error:%v",
			tc.name, tc.queryParams["region"], tc.queryParams["environment"], tc.queryParams["status"], tc.offset, tc.limit, tc.expectedError)

		clusters, count, more, err := db.ListClusters(
			tc.offset,
			tc.limit,
			tc.queryParams["region"],
			tc.queryParams["environment"],
			tc.queryParams["status"])

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
		test.Equal(tc.expectedCount, count)
		test.Equal(tc.expectedMore, more)
	}
}
