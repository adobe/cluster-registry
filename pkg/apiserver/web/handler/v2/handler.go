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

package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/adobe/cluster-registry/pkg/apiserver/models"
	"github.com/adobe/cluster-registry/pkg/k8s"
	"github.com/eko/gocache/lib/v4/cache"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"strconv"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/apiserver/errors"
	"github.com/adobe/cluster-registry/pkg/apiserver/web"
	"github.com/adobe/cluster-registry/pkg/auth"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// Handler interface
type Handler interface {
	GetCluster(echo.Context) error
	PatchCluster(echo.Context) error
	ListClusters(echo.Context) error
	Register(*echo.Group)
}

// ClusterSpec is the struct for updating a cluster's dynamic fields
type ClusterSpec struct {
	Status *string            `json:"status,omitempty" validate:"omitempty,oneof=Inactive Active Deprecated Deleted"`
	Phase  *string            `json:"phase,omitempty" validate:"omitempty,oneof=Building Testing Running Upgrading"`
	Tags   *map[string]string `json:"tags,omitempty" validate:"omitempty"`
}

type ClusterPatch struct {
	Spec ClusterSpec `json:"spec" validate:"required"`
}

// handler struct
type handler struct {
	db        database.Db
	appConfig *config.AppConfig
	metrics   monitoring.MetricsI
	kcp       k8s.ClientProviderI
	cache     *cache.Cache[string]
}

// NewHandler func
func NewHandler(appConfig *config.AppConfig, d database.Db, m monitoring.MetricsI, kcp k8s.ClientProviderI, cache *cache.Cache[string]) Handler {
	h := &handler{
		db:        d,
		metrics:   m,
		appConfig: appConfig,
		kcp:       kcp,
		cache:     cache,
	}
	return h
}

func (h *handler) Register(v2 *echo.Group) {
	a, err := auth.NewAuthenticator(h.appConfig, h.metrics)
	if err != nil {
		log.Fatalf("Failed to initialize authenticator: %v", err)
	}
	clusters := v2.Group("/clusters", a.VerifyToken(), web.RateLimiter(h.appConfig))
	clusters.GET("/:name", h.GetCluster)
	clusters.PATCH("/:name", h.PatchCluster, a.VerifyGroupAccess(h.appConfig.ApiAuthorizedGroupId))
	clusters.GET("", h.ListClusters, web.HTTPCache(h.cache, h.appConfig))

	services := v2.Group("/services", a.VerifyToken(), web.RateLimiter(h.appConfig))
	services.GET("/:serviceId", h.GetServiceMetadata)
	services.GET("/:serviceId/cluster/:clusterName", h.GetServiceMetadataForCluster)
}

// GetCluster godoc
// @Summary Get a cluster
// @Description Get a cluster. Auth is required
// @ID v2-get-cluster
// @Tags cluster
// @Accept  json
// @Produce  json
// @Param name path string true "Name of the cluster to get"
// @Success 200 {object} registryv1.ClusterSpec
// @Failure 400 {object} errors.Error
// @Failure 500 {object} errors.Error
// @Security bearerAuth
// @Router /v2/clusters/{name} [get]
func (h *handler) GetCluster(c echo.Context) error {
	name := c.Param("name")
	cluster, err := h.getCluster(h.db, name)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewError(err))
	}

	if cluster == nil {
		return c.JSON(http.StatusNotFound, errors.NotFound())
	}

	return c.JSON(http.StatusOK, newClusterResponse(cluster))
}

// ListClusters
// @Summary List clusters
// @Description List all or a subset of clusters. Use conditions to filter clusters based on their fields.
// @ID v2-get-clusters
// @Tags cluster
// @Accept  json
// @Produce  json
// @Param conditions query []string false "Filter conditions" collectionFormat(multi)
// @Param offset query integer false "Offset to start pagination search results (default is 0)"
// @Param limit query integer false "The number of results per page (default is 200)"
// @Success 200 {object} clusterList
// @Failure 500 {object} errors.Error
// @Security bearerAuth
// @Router /v2/clusters [get]
func (h *handler) ListClusters(c echo.Context) error {
	var clusters []registryv1.Cluster
	var count int

	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		offset = 0
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 200
	}

	filter := database.NewDynamoDBFilter()
	queryConditions := getQueryConditions(c)

	if len(queryConditions) == 0 {
		clusters, count, more, _ := h.db.ListClusters(offset, limit, "", "", "", "")
		return c.JSON(http.StatusOK, newClusterListResponse(clusters, count, offset, limit, more))
	}

	for _, qc := range queryConditions {
		condition, err := models.NewFilterConditionFromQuery(qc)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errors.NewError(err))
		}
		filter.AddCondition(condition)
	}

	clusters, count, more, _ := h.db.ListClustersWithFilter(offset, limit, filter)
	return c.JSON(http.StatusOK, newClusterListResponse(clusters, count, offset, limit, more))
}

// PatchCluster godoc
// @Summary Patch a cluster
// @Description Update a cluster. Auth is required
// @ID v2-patch-cluster
// @Tags cluster
// @Accept  json
// @Produce  json
// @Param name path string true "Name of the cluster to patch"
// @Param clusterSpec body ClusterSpec true "Request body"
// @Success 200 {object} registryv1.ClusterSpec
// @Failure 400 {object} errors.Error
// @Failure 500 {object} errors.Error
// @Security bearerAuth
// @Router /v2/clusters/{name} [patch]
func (h *handler) PatchCluster(c echo.Context) error {

	name := c.Param("name")
	cluster, err := h.getCluster(h.db, name)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewError(err))
	}

	if cluster == nil {
		return c.JSON(http.StatusNotFound, errors.NotFound())
	}

	var clusterSpec ClusterSpec

	if err = c.Bind(&clusterSpec); err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewError(err))
	}

	if err = clusterSpec.Validate(c); err != nil {
		return c.JSON(http.StatusBadRequest, errors.NewError(err))
	}

	err = h.patchCluster(cluster, clusterSpec)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewError(err))
	}

	return c.JSON(http.StatusOK, newClusterResponse(cluster))
}

// GetServiceMetadata
// @Summary Get service metadata
// @Description List all metadata for a service for all clusters
// @ID v2-get-service-metadata
// @Tags service
// @Accept  json
// @Produce  json
// @Param serviceId path string true "SNOW Service ID"
// @Param conditions query []string false "Filter conditions" collectionFormat(multi)
// @Param offset query integer false "Offset to start pagination search results (default is 0)"
// @Param limit query integer false "The number of results per page (default is 200)"
// @Success 200 {object} clusterList
// @Failure 500 {object} errors.Error
// @Security bearerAuth
// @Router /v2/services/{serviceId} [get]
func (h *handler) GetServiceMetadata(c echo.Context) error {
	var clusters []registryv1.Cluster
	var count int

	serviceId := c.Param("serviceId")

	offset, err := strconv.Atoi(c.QueryParam("offset"))
	if err != nil {
		offset = 0
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 200
	}

	filter := database.NewDynamoDBFilter()
	queryConditions := getQueryConditions(c)

	if len(queryConditions) == 0 {
		clusters, count, more, _ := h.db.ListClustersWithService(serviceId, offset, limit, "", "", "", "")
		return c.JSON(http.StatusOK, newServiceMetadataListResponse(clusters, count, offset, limit, more))
	}

	for _, qc := range queryConditions {
		condition, err := models.NewFilterConditionFromQuery(qc)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errors.NewError(err))
		}
		filter.AddCondition(condition)
	}

	clusters, count, more, _ := h.db.ListClustersWithServiceAndFilter(serviceId, offset, limit, filter)
	return c.JSON(http.StatusOK, newServiceMetadataListResponse(clusters, count, offset, limit, more))
}

// GetServiceMetadataForCluster
// @Summary Get service metadata for a specific cluster
// @Description Get metadata for a service for a specific cluster
// @ID v2-get-service-metadata-for-cluster
// @Tags service
// @Accept  json
// @Produce  json
// @Param serviceId path string true "SNOW Service ID"
// @Param clusterName path string true "Name of the cluster"
// @Success 200 {object} registryv1.ClusterSpec
// @Failure 400 {object} errors.Error
// @Failure 500 {object} errors.Error
// @Security bearerAuth
// @Router /v2/services/{serviceId}/cluster/{clusterName} [get]
func (h *handler) GetServiceMetadataForCluster(c echo.Context) error {
	serviceId := c.Param("serviceId")
	clusterName := c.Param("clusterName")
	cluster, err := h.db.GetClusterWithService(serviceId, clusterName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errors.NewError(err))
	}

	if cluster == nil {
		return c.JSON(http.StatusNotFound, errors.NotFound())
	}

	return c.JSON(http.StatusOK, newServiceMetadataResponse(cluster))
}

// getCluster by standard name or short name
func (h *handler) getCluster(db database.Db, name string) (*registryv1.Cluster, error) {

	var cluster *registryv1.Cluster
	var err error

	cluster, err = db.GetCluster(name)
	if err != nil {
		return nil, err
	}

	if cluster == nil {
		dashName, err := web.GetClusterDashName(name)
		if err != nil {
			log.Warnf("Cluster %s is not a short name. Error: %v", name, err.Error())
		} else {
			cluster, err = db.GetCluster(dashName)
			if err != nil {
				return nil, err
			}
		}
	}
	return cluster, nil
}

// patchCluster
func (h *handler) patchCluster(cluster *registryv1.Cluster, spec ClusterSpec) error {
	client, err := h.kcp.GetClient(h.appConfig, cluster)
	if err != nil {
		return fmt.Errorf("failed to get client for cluster %s: %v", cluster.Spec.Name, err)
	}

	patch, err := json.Marshal(&ClusterPatch{Spec: spec})
	if err != nil {
		return err
	}

	res, err := client.CoreV1().RESTClient().
		Patch(types.MergePatchType).
		AbsPath("/apis/registry.ethos.adobe.com/v1").
		Namespace("cluster-registry").
		Resource("clusters").
		Name(cluster.Spec.Name).
		Body(patch).
		DoRaw(context.TODO())

	log.Debugf("Patch response: %s", string(res))

	if err != nil {
		return err
	}

	return nil
}

func getQueryConditions(c echo.Context) []string {
	for k, v := range c.QueryParams() {
		if k == "conditions" {
			return v
		}
	}
	return []string{}
}

func (patch *ClusterSpec) Validate(c echo.Context) error {
	if err := c.Validate(patch); err != nil {
		return err
	}

	if patch.Tags != nil && len(*patch.Tags) > 0 {
		for key, value := range *patch.Tags {
			if err := validateTag(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateTag(key, value string) error {
	switch key {
	case "onboarding", "scaling":
		if value != "on" && value != "off" {
			return fmt.Errorf("%s tag value must be 'on' or 'off'", key)
		}
	default:
		return fmt.Errorf("invalid tag %s", key)
	}
	return nil
}
