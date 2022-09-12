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

/*
This is a service that runs the SLT check and saves the metrics for prometheus.
*/

package main

import (
	"net/http"

	"github.com/adobe/cluster-registry/test/slt/checks"
	"github.com/adobe/cluster-registry/test/slt/checks/request"
	"github.com/adobe/cluster-registry/test/slt/checks/update"
	h "github.com/adobe/cluster-registry/test/slt/helpers"
	log "github.com/adobe/cluster-registry/test/slt/log"
	web "github.com/adobe/cluster-registry/test/slt/web"
)

var (
	logger                    *log.Logger
	timeBetweenE2e            string // Time to wait between e2e test
	timeBetweenGetCluster     string // Time to wait between get cluster test
	timeBetweenGetAllClusters string // Time to wait between get clusters test
	tokenRefreshTime          string // The time between token refreshes
)

// Runs before main
func init() {
	logger = log.NewLogger("slt-service")

	checks.SetLogger(logger)

	timeBetweenE2e = h.GetEnv("TIME_BETWEEN_E2E", "2m", logger)
	timeBetweenGetCluster = h.GetEnv("TIME_BETWEEN_GET_CLUSTER", "1m", logger)
	timeBetweenGetAllClusters = h.GetEnv("TIME_BETWEEN_GET_ALL_CLUSTERS", "1m", logger)
	tokenRefreshTime = h.GetEnv("TOKEN_REFRESH_TIME", "29m", logger)

	go h.RunFuncInLoop(checks.RefreshToken, nil, tokenRefreshTime, "", logger)
}

func main() {
	e := web.NewEchoWithLogger(logger)

	e.GET("/metrics", web.Metrics())
	e.GET("/livez", web.Livez)

	go func() {
		if err := e.Start(":8081"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalf("shutting down the server: %s", err.Error())
		}
	}()

	go h.RunFuncInLoop(
		checks.RunE2eTest,
		update.GetConfigFromEnv(),
		timeBetweenE2e,
		"5s",
		logger,
	)

	go h.RunFuncInLoop(
		checks.RunClusterRequest,
		request.GetClusterConfigFromEnv(),
		timeBetweenGetCluster,
		"15s",
		logger,
	)

	go h.RunFuncInLoop(
		checks.RunAllClustersRequests,
		request.GetAllClusterConfigFromEnv(),
		timeBetweenGetAllClusters,
		"20s",
		logger,
	)

	web.WaitForShutdownSignal(e)
}
