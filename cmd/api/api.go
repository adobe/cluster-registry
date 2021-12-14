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
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Swagger Example API
// @version 1.0
// @description Cluster Registry API
// @title Cluster Registry API

// @host http://cluster-registry.missionctrl.cloud.adobe.io
// @BasePath /api

// @schemes http https
// @produce	application/json
// @consumes application/json

// @securityDefinitions.apikey bearerAuth
// @in header
// @name Authorization
func main() {
	a := api.NewRouter()

	a.GET("/api/swagger/*", echoSwagger.WrapHandler)
	a.GET("/livez", monitoring.Livez)

	v1 := a.Group("/api/v1")
	m := monitoring.NewMetrics("cluster_registry_api", nil, false)
	m.Use(a)

	d := database.NewDb(m)
	h := api.NewHandler(d, m)
	h.Register(v1)

	c := sqs.NewConsumer(d, m)
	go c.Consume()

	a.Logger.Fatal(a.Start(":8080"))
}
