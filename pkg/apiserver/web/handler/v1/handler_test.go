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
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	redisstore "github.com/eko/gocache/store/redis/v4"
	"github.com/go-redis/redismock/v9"
	"github.com/gusaul/go-dynamock"
	"github.com/redis/go-redis/v9"
	"net/http"
	"net/http/httptest"
	"testing"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/apiserver/web"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

var (
	appConfig    *config.AppConfig
	m            *monitoring.Metrics
	db           database.Db
	dbMock       *dynamock.DynaMock
	redisClient  *redis.Client
	redisMock    redismock.ClientMock
	redisStore   *redisstore.RedisStore
	cacheManager *cache.Cache[string]
)

// mockDatabase extends database.db
type mockDatabase struct {
	database.Db
	clusters []registryv1.Cluster
}

func init() {
	appConfig = &config.AppConfig{}
	m = monitoring.NewMetrics("cluster_registry_api_handler_test", true)
	db = database.NewDb(appConfig, m)
	dbMock = db.Mock()
	redisClient, redisMock = redismock.NewClientMock()
	redisStore = redisstore.NewRedis(redisClient)
	cacheManager = cache.New[string](redisStore)
}

func (m mockDatabase) GetCluster(name string) (*registryv1.Cluster, error) {
	for _, c := range m.clusters {
		if c.Spec.Name == name {
			return &c, nil
		}
	}
	return nil, nil
}

func (m mockDatabase) ListClusters(offset int, limit int, environment string, region string, status string, lastUpdated string) ([]registryv1.Cluster, int, bool, error) {
	return m.clusters, len(m.clusters), false, nil
}

func TestNewHandler(t *testing.T) {
	test := assert.New(t)
	appConfig := &config.AppConfig{}
	d := mockDatabase{}
	m := monitoring.NewMetrics("cluster_registry_api_handler_test", true)
	h := NewHandler(appConfig, d, m, cacheManager)
	test.NotNil(h)
}

func TestGetCluster(t *testing.T) {
	test := assert.New(t)
	appConfig := &config.AppConfig{}

	t.Log("Test getting a single cluster from the api.")

	tcs := []struct {
		name             string
		clusterName      string
		clusters         []registryv1.Cluster
		expectedCluster  registryv1.Cluster
		expectedResponse string
		expectedStatus   int
	}{
		{
			name:        "get existing cluster",
			clusterName: "cluster1",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}},
			expectedCluster: registryv1.Cluster{
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
			name:        "get nonexistent cluster",
			clusterName: "cluster2",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}},
			expectedCluster: registryv1.Cluster{},
			expectedStatus:  http.StatusNotFound,
		},
		{
			name:        "get cluster by shortname",
			clusterName: "cluster1produseast1",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1-prod-useast1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}},
			expectedCluster: registryv1.Cluster{
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

		d := mockDatabase{clusters: tc.clusters}
		m := monitoring.NewMetrics("cluster_registry_api_handler_test", true)
		r := web.NewRouter()
		h := NewHandler(appConfig, d, m, cacheManager)

		req := httptest.NewRequest(echo.GET, "/api/v1/clusters/:name", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		ctx := r.NewContext(req, rec)
		ctx.SetPath("/api/articles/:name")
		ctx.SetParamNames("name")
		ctx.SetParamValues(tc.clusterName)

		t.Logf("\tTest %s:\tWhen checking for cluster %s and http status code %d", tc.name, tc.clusterName, tc.expectedStatus)

		err := h.GetCluster(ctx)
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

	t.Log("Test getting all clusters from the api.")

	tcs := []struct {
		name           string
		clusters       []registryv1.Cluster
		expectedStatus int
		expectedItems  int
	}{
		{
			name: "get all clusters",
			clusters: []registryv1.Cluster{{
				Spec: registryv1.ClusterSpec{
					Name:         "cluster1",
					LastUpdated:  "2020-02-14T06:15:32Z",
					RegisteredAt: "2019-02-14T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "off"},
				}}, {
				Spec: registryv1.ClusterSpec{
					Name:         "cluster2",
					LastUpdated:  "2020-02-13T06:15:32Z",
					RegisteredAt: "2019-02-13T06:15:32Z",
					Status:       "Active",
					Phase:        "Running",
					Tags:         map[string]string{"onboarding": "on", "scaling": "on"},
				}}},
			expectedStatus: http.StatusOK,
			expectedItems:  2,
		},
	}
	for _, tc := range tcs {

		d := mockDatabase{clusters: tc.clusters}
		m := monitoring.NewMetrics("cluster_registry_api_handler_test", true)
		r := web.NewRouter()
		h := NewHandler(appConfig, d, m, cacheManager)

		req := httptest.NewRequest(echo.GET, "/api/v1/clusters", nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		ctx := r.NewContext(req, rec)

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

func TestListClustersWithEmptyCache(t *testing.T) {
	test := assert.New(t)

	t.Log("Test caching cluster list results.")

	redisMock.MatchExpectationsInOrder(true)

	expectedClusters := []registryv1.Cluster{
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
	}

	var expectedItems []map[string]*dynamodb.AttributeValue
	for _, c := range expectedClusters {
		item, err := dynamodbattribute.MarshalMap(database.ClusterDb{
			Cluster: &c,
		})
		test.NoError(err)
		expectedItems = append(expectedItems, item)
	}

	expectedResult := dynamodb.QueryOutput{
		Items: expectedItems,
	}
	dbMock.ExpectQuery().WillReturns(expectedResult)

	r := web.NewRouter()
	h := NewHandler(appConfig, db, m, cacheManager)

	req := httptest.NewRequest(echo.GET, "/api/v1/clusters", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := r.NewContext(req, rec)

	expectedClusterListResponse := newClusterListResponse(expectedClusters, len(expectedClusters), 0, 200, false)
	expectedBody, err := json.Marshal(expectedClusterListResponse)
	test.NoError(err)

	expectedBody = append(expectedBody, "\n"...)

	expectedResponse := web.Response{
		Value:  expectedBody,
		Header: http.Header{"Content-Type": []string{echo.MIMEApplicationJSON}},
	}

	key := web.GenerateKey(ctx.Request().URL.String())

	redisMock.ExpectGet(key).SetVal("")

	redisMock.ExpectSet(key, expectedResponse.String(), appConfig.ApiCacheTTL).SetVal("OK")

	err = web.HTTPCache(cacheManager, appConfig, []string{"clusters"})(h.ListClusters)(ctx)
	test.NoError(err)

	if err := redisMock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestListClustersWithCache(t *testing.T) {
	test := assert.New(t)

	t.Log("Test caching cluster list results.")

	redisMock.MatchExpectationsInOrder(true)

	expectedClusters := []registryv1.Cluster{
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
	}

	r := web.NewRouter()
	h := NewHandler(appConfig, db, m, cacheManager)

	req := httptest.NewRequest(echo.GET, "/api/v1/clusters", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := r.NewContext(req, rec)

	expectedClusterListResponse := newClusterListResponse(expectedClusters, len(expectedClusters), 0, 200, false)
	expectedBody, err := json.Marshal(expectedClusterListResponse)
	test.NoError(err)

	expectedBody = append(expectedBody, "\n"...)

	expectedResponse := web.Response{
		Value:  expectedBody,
		Header: http.Header{"Content-Type": []string{echo.MIMEApplicationJSON}},
	}

	key := web.GenerateKey(ctx.Request().URL.String())

	redisMock.ExpectSet(key, expectedResponse.String(), appConfig.ApiCacheTTL).SetVal("")
	err = redisStore.Set(context.Background(), key, expectedResponse.String(), store.WithExpiration(appConfig.ApiCacheTTL))
	test.NoError(err)

	redisMock.ExpectGet(key).SetVal(string(expectedBody))

	err = web.HTTPCache(cacheManager, appConfig, []string{"clusters"})(h.ListClusters)(ctx)
	test.NoError(err)

	if err := redisMock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
