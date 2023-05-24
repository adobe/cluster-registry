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

package v2

import (
	"encoding/json"
	"fmt"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/apiserver/web"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gusaul/go-dynamock"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	appConfig *config.AppConfig
	m         *monitoring.Metrics
	db        database.Db
	dbMock    *dynamock.DynaMock
)

func init() {
	appConfig = &config.AppConfig{}
	m = monitoring.NewMetrics("cluster_registry_api_handler_test", true)
	db = database.NewDb(appConfig, m)
	dbMock = db.Mock()
}

func TestNewHandler(t *testing.T) {
	test := assert.New(t)
	h := NewHandler(appConfig, db, m)
	test.NotNil(h)
}

func TestGetCluster(t *testing.T) {
	test := assert.New(t)

	t.Log("Test getting a single cluster from the api.")

	tcs := []struct {
		name            string
		clusterName     string
		expectedCluster *registryv1.Cluster
		expectedStatus  int
	}{
		{
			name:        "get existing cluster",
			clusterName: "cluster1",
			expectedCluster: &registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			expectedStatus: http.StatusOK,
		},
		{
			name:            "get nonexistent cluster",
			clusterName:     "cluster2",
			expectedCluster: nil,
			expectedStatus:  http.StatusNotFound,
		},
		{
			name:        "get cluster by shortname",
			clusterName: "cluster1produseast1",
			expectedCluster: &registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1-prod-useast1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tcs {
		r := web.NewRouter()
		h := NewHandler(appConfig, db, m)

		req := httptest.NewRequest(echo.GET, "/api/v2/clusters/:name", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		ctx := r.NewContext(req, rec)
		ctx.SetPath("/api/v2/clusters/:name")
		ctx.SetParamNames("name")
		ctx.SetParamValues(tc.clusterName)

		expectedItem, err := dynamodbattribute.MarshalMap(
			database.ClusterDb{
				Cluster: tc.expectedCluster,
			})
		test.NoError(err)

		expectedResult := dynamodb.GetItemOutput{
			Item: expectedItem,
		}
		dbMock.ExpectGetItem().WillReturns(expectedResult)

		t.Logf("\tTest %s:\tWhen checking for cluster %s and http status code %d", tc.name, tc.clusterName, tc.expectedStatus)

		err = h.GetCluster(ctx)
		test.NoError(err)

		test.Equal(tc.expectedStatus, rec.Code)
		if rec.Code == http.StatusOK {
			var c registryv1.ClusterSpec
			err := json.Unmarshal(rec.Body.Bytes(), &c)
			test.NoError(err)
			test.Equal(tc.expectedCluster.Spec.Name, c.Name)
		}
	}
}

func TestListClusters(t *testing.T) {
	test := assert.New(t)
	appConfig := &config.AppConfig{}

	t.Log("Test getting multiple clusters from the api.")

	tcs := []struct {
		name             string
		filter           []string
		expectedClusters []registryv1.Cluster
		expectedStatus   int
		expectedItems    int
	}{
		{
			name:   "get all clusters",
			filter: []string{},
			expectedClusters: []registryv1.Cluster{
				{
					Spec: registryv1.ClusterSpec{
						Name:         "cluster1",
						LastUpdated:  "2020-02-14T06:15:32Z",
						RegisteredAt: "2019-02-14T06:15:32Z",
						Status:       "Active",
						Phase:        "Running",
						Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
					},
				},
				{
					Spec: registryv1.ClusterSpec{
						Name:         "cluster2",
						LastUpdated:  "2020-03-14T06:15:32Z",
						RegisteredAt: "2019-03-14T06:15:32Z",
						Status:       "Active",
						Phase:        "Upgrading",
						Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedItems:  2,
		},
		{
			name: "get subset of clusters clusters",
			filter: []string{
				"status:=Active",
				"phase:=Running",
			},
			expectedClusters: []registryv1.Cluster{
				{
					Spec: registryv1.ClusterSpec{
						Name:         "cluster1",
						LastUpdated:  "2020-02-14T06:15:32Z",
						RegisteredAt: "2019-02-14T06:15:32Z",
						Status:       "Active",
						Phase:        "Running",
						Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
					},
				},
			},
			expectedStatus: http.StatusOK,
			expectedItems:  1,
		},
	}
	for _, tc := range tcs {
		r := web.NewRouter()
		h := NewHandler(appConfig, db, m)

		for i, v := range tc.filter {
			tc.filter[i] = fmt.Sprintf("conditions=%s", v)
		}
		filterQuery := strings.Join(tc.filter, "&")

		req := httptest.NewRequest(echo.GET, fmt.Sprintf("/api/v2/clusters?%s", filterQuery), nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		ctx := r.NewContext(req, rec)

		var expectedItems []map[string]*dynamodb.AttributeValue
		for _, c := range tc.expectedClusters {
			item, err := dynamodbattribute.MarshalMap(database.ClusterDb{
				Cluster: &c,
			})
			test.NoError(err)
			expectedItems = append(expectedItems, item)
		}

		if len(tc.filter) > 0 {
			expectedResult := dynamodb.ScanOutput{
				Items: expectedItems,
			}
			dbMock.ExpectScan().WillReturns(expectedResult)
		} else {
			expectedResult := dynamodb.QueryOutput{
				Items: expectedItems,
			}
			dbMock.ExpectQuery().WillReturns(expectedResult)
		}

		t.Logf("\tTest %s:\tWhen checking for status code %d and number of items %d", tc.name, tc.expectedStatus, tc.expectedItems)

		err := h.ListClusters(ctx)

		test.NoError(err)
		test.Equal(tc.expectedStatus, rec.Code)

		if rec.Code == http.StatusOK {
			var cl clusterList
			err := json.Unmarshal(rec.Body.Bytes(), &cl)

			test.NoError(err)
			test.Equal(tc.expectedItems, cl.ItemsCount)
		}
	}
}

func TestPatchCluster(t *testing.T) {
	test := assert.New(t)

	t.Log("Test patching a cluster.")

	tcs := []struct {
		name            string
		clusterName     string
		clusterPatch    ClusterPatch
		expectedCluster *registryv1.Cluster
		expectedStatus  int
	}{
		{
			name:        "patch status existing cluster",
			clusterName: "cluster1",
			clusterPatch: ClusterPatch{
				Status: "Active",
			},
			expectedCluster: &registryv1.Cluster{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "patch non-existing cluster",
			clusterName: "cluster2",
			clusterPatch: ClusterPatch{
				Status: "Active",
			},
			expectedCluster: nil,
			expectedStatus:  http.StatusNotFound,
		},
	}

	for _, tc := range tcs {
		r := web.NewRouter()
		h := NewHandler(appConfig, db, m)

		patch, err := json.Marshal(tc.clusterPatch)
		body := strings.NewReader(string(patch))
		req := httptest.NewRequest(echo.PATCH, "/api/v2/clusters/:name", body)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		ctx := r.NewContext(req, rec)
		ctx.SetPath("/api/v2/clusters/:name")
		ctx.SetParamNames("name")
		ctx.SetParamValues(tc.clusterName)

		expectedItem, err := dynamodbattribute.MarshalMap(
			database.ClusterDb{
				Cluster: tc.expectedCluster,
			})
		test.NoError(err)

		expectedResult := dynamodb.GetItemOutput{
			Item: expectedItem,
		}
		dbMock.ExpectGetItem().WillReturns(expectedResult)

		t.Logf("\tTest %s:\tWhen checking for cluster %s and http status code %d", tc.name, tc.clusterName, tc.expectedStatus)

		err = h.GetCluster(ctx)
		test.NoError(err)

		test.Equal(tc.expectedStatus, rec.Code)
	}
}
