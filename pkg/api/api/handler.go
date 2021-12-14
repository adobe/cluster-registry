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

package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/adobe/cluster-registry/pkg/api/authz"
	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// Handler interface
type Handler interface {
	GetCluster(echo.Context) error
	ListClusters(echo.Context) error
	Register(*echo.Group)
}

// handler struct
type handler struct {
	db  database.Db
	met monitoring.MetricsI
}

// NewHandler func
func NewHandler(d database.Db, m monitoring.MetricsI) Handler {
	h := &handler{
		db:  d,
		met: m,
	}
	return h
}

func (h *handler) Register(v1 *echo.Group) {
	a, err := authz.NewAuthenticator(h.met)
	if err != nil {
		log.Fatalf("Failed to initialize authenticator: %v", err)
	}
	clusters := v1.Group("/clusters", a.VerifyToken())
	clusters.GET("/:name", h.GetCluster)
	clusters.GET("", h.ListClusters)
}

// GetCluster godoc
// @Summary Get an cluster
// @Description Get an cluster. Auth is required
// @ID get-cluster
// @Tags cluster
// @Accept  json
// @Produce  json
// @Param name path string true "Name of the cluster to get"
// @Success 200 {object} registryv1.ClusterSpec
// @Failure 400 {object} utils.Error
// @Failure 500 {object} utils.Error
// @Security bearerAuth
// @Router /v1/clusters/{name} [get]
func (h *handler) GetCluster(ctx echo.Context) error {
	name := ctx.Param("name")
	c, err := h.db.GetCluster(name)

	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, utils.NewError(err))
	}

	if c == nil {
		return ctx.JSON(http.StatusNotFound, utils.NotFound())
	}

	return ctx.JSON(http.StatusOK, newClusterResponse(ctx, c))
}

// ListClusters godoc
// @Summary List all clusters
// @Description List all clusters. Use query parameters to filter results. Auth is required
// @ID get-clusters
// @Tags cluster
// @Accept  json
// @Produce  json
// @Param region query string false "Filter by region"
// @Param environment query string false "Filter by environment"
// @Param businessUnit query string false "Filter by businessUnit"
// @Param status query string false "Filter by status"
// @Param limit query integer false "Limit number of clusters returned (default is 10)"
// @Param offset query integer false "Offset/skip number of clusters (default is 0)"
// @Success 200 {object} clusterList
// @Failure 500 {object} utils.Error
// @Security bearerAuth
// @Router /v1/clusters [get]
func (h *handler) ListClusters(ctx echo.Context) error {
	var (
		clusters []registryv1.Cluster
		count    int
	)

	region := ctx.QueryParam("region")
	environment := ctx.QueryParam("environment")
	businessUnit := ctx.QueryParam("businessUnit")
	status := ctx.QueryParam("status")

	offset, err := strconv.Atoi(ctx.QueryParam("offset"))
	if err != nil {
		offset = 0
	}

	limit, err := strconv.Atoi(ctx.QueryParam("limit"))
	if err != nil {
		limit = 20
	}

	fmt.Println(limit, offset)
	clusters, count, _ = h.db.ListClusters(region, environment, businessUnit, status)

	return ctx.JSON(http.StatusOK, newClusterListResponse(clusters, count))
}
