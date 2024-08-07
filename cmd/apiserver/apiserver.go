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

package main

import (
	"context"
	"github.com/adobe/cluster-registry/pkg/apiserver/docs"
	"github.com/adobe/cluster-registry/pkg/apiserver/event"
	"github.com/adobe/cluster-registry/pkg/apiserver/web"
	api "github.com/adobe/cluster-registry/pkg/apiserver/web"
	apiv1 "github.com/adobe/cluster-registry/pkg/apiserver/web/handler/v1"
	apiv2 "github.com/adobe/cluster-registry/pkg/apiserver/web/handler/v2"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	"github.com/adobe/cluster-registry/pkg/k8s"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/adobe/cluster-registry/pkg/sqs"
	awssqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	redisstore "github.com/eko/gocache/store/redis/v4"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// Version it's passed as ldflags in the build process
var Version = "dev"

// @title Cluster Registry API
// @version 1.0
// @description Cluster Registry API

// @host 127.0.0.1:8080
// @BasePath /api

// @schemes http https
// @produce	application/json
// @consumes application/json

// @securityDefinitions.apikey bearerAuth
// @in header
// @name Authorization
func main() {

	appConfig, err := config.LoadApiConfig()
	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
		return
	}

	log.SetLevel(appConfig.LogLevel)
	log.Debugf("Config loaded successfully %+v:", appConfig)

	m := monitoring.NewMetrics("cluster_registry_api", false)
	db := database.NewDb(appConfig, m)
	q, err := sqs.NewSQS(sqs.Config{
		AWSRegion:         appConfig.SqsAwsRegion,
		Endpoint:          appConfig.SqsEndpoint,
		QueueName:         appConfig.SqsQueueName,
		BatchSize:         appConfig.SqsBatchSize,
		VisibilityTimeout: 120,
		WaitSeconds:       appConfig.SqsWaitSeconds,
		RunInterval:       appConfig.SqsRunInterval,
		RunOnce:           false,
		MaxHandlers:       10,
		BusyTimeout:       30,
	})

	if err != nil {
		log.Fatalf("Cannot create SQS client: %s", err.Error())
		return
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: appConfig.ApiCacheRedisHost,
	})
	cmd := redisClient.Info(context.Background())
	if cmd.Err() != nil {
		log.Fatalf("Cannot connect to redis: %s", cmd.Err().Error())
		return
	}
	redisStore := redisstore.NewRedis(redisClient)
	cacheManager := cache.New[string](redisStore)

	handler := event.NewClusterUpdateHandler(db)
	q.RegisterHandler(func(msg *awssqs.Message) {
		log.Debugf("Received message: %s", *msg.MessageId)
		e, err := sqs.NewEvent(msg)
		if err != nil {
			log.Errorf("Cannot create event from message: %s", err.Error())
			return
		}
		if e.Type != sqs.ClusterUpdateEvent {
			log.Infof("Not interested in event of type %s, skipping", e.Type)
			return
		}
		log.Debugf("Handling event for message: %s", *msg.MessageId)
		if err = handler.Handle(e); err != nil {
			log.Errorf("Failed to handle event: %s", err.Error())
			return
		}
		if err = q.Delete(msg); err != nil {
			log.Errorf("Failed to delete message: %s", err.Error())
			return
		}
		log.Debugf("Invalidating clusters cache")
		err = cacheManager.Invalidate(context.Background(), store.WithInvalidateTags([]string{"clusters"}))
		if err != nil {
			log.Errorf("Failed to invalidate clusters cache: %s", err.Error())
			return
		}
	})

	a := api.NewRouter()
	status := api.StatusSessions{
		Db:        db,
		SQS:       q,
		AppConfig: appConfig,
	}

	docs.SwaggerInfo.Host = appConfig.ApiHost

	a.GET("/api/swagger/*", echoSwagger.WrapHandler)
	a.GET("/livez", web.Livez)
	a.GET("/readyz", status.Readyz)
	a.GET("/version", web.Version(Version))
	a.GET("/metrics", web.Metrics())

	v1 := a.Group("/api/v1")
	hv1 := apiv1.NewHandler(appConfig, db, m, cacheManager)
	hv1.Register(v1)

	v2 := a.Group("/api/v2")
	hv2 := apiv2.NewHandler(appConfig, db, m, &k8s.ClientProvider{}, cacheManager)
	hv2.Register(v2)

	go q.Poll()

	m.Use(a)
	a.Logger.Fatal(a.Start(":8080"))
}
