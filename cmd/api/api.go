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

package main

import (
	"github.com/adobe/cluster-registry/pkg/api/api"
	"github.com/adobe/cluster-registry/pkg/api/database"
	_ "github.com/adobe/cluster-registry/pkg/api/docs"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/sqs"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	"github.com/labstack/gommon/log"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Swagger Example API
// @version 1.0
// @description Cluster Registry API
// @title Cluster Registry API

// @host http://127.0.0.1:8080
// @BasePath /api

// @schemes http https
// @produce	application/json
// @consumes application/json

// @securityDefinitions.apikey bearerAuth
// @in header
// @name Authorization
func main() {

	appConfig, err := utils.LoadApiConfig()
	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	a := api.NewRouter()

	a.GET("/api/swagger/*", echoSwagger.WrapHandler)
	a.GET("/livez", monitoring.Livez)

	v1 := a.Group("/api/v1")
	m := monitoring.NewMetrics("cluster_registry_api", nil, false)
	m.Use(a)

	dbHandle := database.NewDb(appConfig, m)

	h := api.NewHandler(appConfig, dbHandle, m)
	h.Register(v1)

	sqsHandle := sqs.NewSQS(appConfig)
	c := sqs.NewConsumer(sqsHandle, appConfig, dbHandle, m)
	go c.Consume()

	status := api.StatusSessions{
		Db:        dbHandle,
		Consumer:  c,
		AppConfig: appConfig,
	}
	a.GET("/status", status.ServiceStatus)

	a.Logger.Fatal(a.Start(":8080"))
}
