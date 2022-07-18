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
	// docs "github.com/adobe/cluster-registry/pkg/apiserver/docs"
	"github.com/adobe/cluster-registry/pkg/apiserver/docs"
	"github.com/adobe/cluster-registry/pkg/apiserver/web"
	api "github.com/adobe/cluster-registry/pkg/apiserver/web"
	apiv1 "github.com/adobe/cluster-registry/pkg/apiserver/web/handler/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"github.com/labstack/gommon/log"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Version it's passed as ldflags in the build process
var Version = "dev"

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

	appConfig, err := config.LoadApiConfig()
	log.SetLevel(appConfig.LogLevel)

	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
	} else {
		log.Debugf("Config loaded succesfully %+v:", appConfig)
	}

	m := monitoring.NewMetrics("cluster_registry_api", false)
	db := database.NewDb(appConfig, m)
	s := sqs.NewSQS(appConfig)
	c := sqs.NewConsumer(s, appConfig, db, m)
	a := api.NewRouter()
	status := api.StatusSessions{
		Db:        db,
		Consumer:  c,
		AppConfig: appConfig,
	}

	docs.SwaggerInfo.Host = appConfig.ApiHost

	a.GET("/api/swagger/*", echoSwagger.WrapHandler)
	a.GET("/livez", web.Livez)
	a.GET("/readyz", status.Readyz)
	a.GET("/version", web.Version(Version))
	a.GET("/metrics", web.Metrics())

	v1 := a.Group("/api/v1")
	hv1 := apiv1.NewHandler(appConfig, db, m)
	hv1.Register(v1)

	go c.Consume()

	m.Use(a)
	a.Logger.Fatal(a.Start(":8080"))
}
