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

package request

import (
	"github.com/labstack/echo/v4"

	h "github.com/adobe/cluster-registry/test/slt/helpers"
	"github.com/adobe/cluster-registry/test/slt/metrics"
)

// MetricLabelGetCluster is the label name for metrics regarding this package
const MetricLabelGetCluster = "get_cluster_test"

// MetricLabelGetAllClusters is the label name for metrics regarding this package
const MetricLabelGetAllClusters = "get_multiple_clusters_test"

var logger echo.Logger

// GetClusterConfig sets the parameters for the GET cluster
type GetClusterConfig struct {
	url         string
	clusterName string
}

// GetAllClusterConfig sets the parameters for the GET cluster
type GetAllClusterConfig struct {
	url          string
	pageNr       string
	perPageLimit string
}

// SetLogger sets the global logger for the slt package
func SetLogger(lgr echo.Logger) {
	logger = lgr
}

// InitMetrics initializes the error metrics to 0
func InitMetrics() {
	metrics.ErrCnt.WithLabelValues(MetricLabelGetCluster).Add(0)
	metrics.ErrCnt.WithLabelValues(MetricLabelGetAllClusters).Add(0)

	metrics.TestStatus.WithLabelValues(MetricLabelGetCluster).Set(0)
	metrics.TestStatus.WithLabelValues(MetricLabelGetAllClusters).Set(0)
}

// GetClusterConfigFromEnv gets from the env the needed global env
func GetClusterConfigFromEnv() GetClusterConfig {
	return GetClusterConfig{
		url:         h.GetEnv("URL", "http://localhost:8080", logger),
		clusterName: h.GetEnv("CLUSTER_NAME", "", logger),
	}
}

// GetAllClusterConfigFromEnv gets from the env the needed global env
func GetAllClusterConfigFromEnv() GetAllClusterConfig {
	return GetAllClusterConfig{
		url:          h.GetEnv("URL", "http://localhost:8080", logger),
		pageNr:       h.GetEnv("PAGE_NR", "0", logger),
		perPageLimit: h.GetEnv("PER_PAGE_LIMIT", "200", logger),
	}
}

// RunGetCluster runs the test
func RunGetCluster(config GetClusterConfig, jwtToken string) error {
	_, err := h.GetCluster(config.url, config.clusterName, jwtToken)
	if err != nil {
		return err
	}
	return nil
}

// RunGetAllClusters runs the test
func RunGetAllClusters(config GetAllClusterConfig, jwtToken string) error {
	_, err := h.GetClusters(config.url, config.perPageLimit, config.pageNr, jwtToken)
	if err != nil {
		return err
	}
	return nil
}
