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

package database

import (
	"context"
	"fmt"
	"github.com/gusaul/go-dynamock"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/labstack/gommon/log"
)

const (
	egressTarget = "database"
)

// Db provides an interface for interacting with dynamoDb
type Db interface {
	GetCluster(name string) (*registryv1.Cluster, error)
	ListClusters(offset int, limit int, environment string, region string, status string, lastUpdated string) ([]registryv1.Cluster, int, bool, error)
	ListClustersWithFilter(offset int, limit int, filter *DynamoDBFilter) ([]registryv1.Cluster, int, bool, error)
	PutCluster(cluster *registryv1.Cluster) error
	DeleteCluster(name string) error
	Status() error
	Mock() *dynamock.DynaMock
	ListClustersWithService(serviceId string, offset int, limit int, environment string, region string, status string, lastUpdated string) ([]registryv1.Cluster, int, bool, error)
	ListClustersWithServiceAndFilter(serviceId string, offset int, limit int, filter *DynamoDBFilter) ([]registryv1.Cluster, int, bool, error)
	GetClusterWithService(serviceId string, clusterName string) (*registryv1.Cluster, error)
}

// db struct
type db struct {
	dbAPI   dynamodbiface.DynamoDBAPI
	table   dbTable
	index   dbTable
	metrics monitoring.MetricsI
}

type dbTable struct {
	name         string
	partitionKey string
	searchKey    string
}

// ClusterDb encapsulates the Cluster CRD
type ClusterDb struct {
	TablePartitionKey string              `json:"name"`
	IndexPartitionKey string              `json:"kind"`
	Region            string              `json:"region"`
	Environment       string              `json:"environment"`
	Status            string              `json:"status"`
	LastUpdatedUnix   int64               `json:"lastUpdatedUnix"`
	Cluster           *registryv1.Cluster `json:"crd"`
}

// NewDb func
func NewDb(appConfig *config.AppConfig, m monitoring.MetricsI) Db {
	var t, i dbTable

	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String(appConfig.AwsRegion),
		Endpoint: aws.String(appConfig.DbEndpoint),
	}))

	t = dbTable{
		name:         appConfig.DbTableName,
		partitionKey: "name",
	}

	i = dbTable{
		name:         appConfig.DbIndexName,
		partitionKey: "kind",
		searchKey:    "name",
	}

	d := dynamodb.New(sess)
	dbInst := &db{
		dbAPI:   d,
		table:   t,
		index:   i,
		metrics: m,
	}

	return dbInst
}

func (d *db) Mock() *dynamock.DynaMock {
	var mock *dynamock.DynaMock
	d.dbAPI, mock = dynamock.New()
	return mock
}

// Status checks if the database is reachable with a 5 sec timeout
func (d *db) Status() error {
	params := &dynamodb.DescribeTableInput{
		TableName: &d.table.name,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := d.dbAPI.DescribeTableWithContext(ctx, params)
	if err != nil {
		d.metrics.RecordErrorCnt(egressTarget)
		msg := fmt.Sprintf("Connectivity check using DescribeTable failed. Error: '%v'", err.Error())
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}

	return nil
}

// GetCluster a single cluster
func (d *db) GetCluster(name string) (*registryv1.Cluster, error) {
	params := &dynamodb.GetItemInput{
		TableName: &d.table.name,
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	}

	start := time.Now()
	resp, err := d.dbAPI.GetItem(params)
	elapsed := float64(time.Since(start)) / float64(time.Second)

	d.metrics.RecordEgressRequestCnt(egressTarget)
	d.metrics.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		msg := fmt.Sprintf("Cannot get cluster '%s' from the database. Error: '%v'", name, err.Error())
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}

	if resp.Item == nil {
		log.Warnf("Cluster '%s' not found in the database.", name)
		return nil, nil
	}

	var clusterDb *ClusterDb
	err = dynamodbattribute.UnmarshalMap(resp.Item, &clusterDb)
	if err != nil {
		msg := fmt.Sprintf("Cannot unmarshal cluster '%s': '%v'", name, err.Error())
		log.Errorf(msg)
		return nil, fmt.Errorf(msg)
	}

	return clusterDb.Cluster, err
}

// ListClusters list all clusters
func (d *db) ListClusters(offset int, limit int, region string, environment string, status string, lastUpdated string) ([]registryv1.Cluster, int, bool, error) {

	var clusters []registryv1.Cluster = []registryv1.Cluster{}
	var queryInput *dynamodb.QueryInput
	var filter expression.ConditionBuilder
	var keyCondition expression.KeyConditionBuilder
	var expr expression.Expression
	var err error

	if status != "" {
		filter = expression.Name("status").Equal(expression.Value(status))
	} else {
		filter = expression.Name("status").NotEqual(expression.Value("Deleted"))
	}

	if region != "" {
		filter = filter.And(expression.Name("region").Equal(expression.Value(region)))
	}

	if environment != "" {
		filter = filter.And(expression.Name("environment").Equal(expression.Value(environment)))
	}

	if lastUpdated != "" {
		t, err := time.Parse(time.RFC3339, lastUpdated)
		if err != nil {
			msg := fmt.Sprintf("Error converting lastUpdated parameter to RFC3339: '%v'.", err)
			log.Errorf(msg)
			return nil, 0, false, fmt.Errorf(msg)
		}
		filter = filter.And(expression.Name("lastUpdatedUnix").GreaterThanEqual(expression.Value(t.Unix())))
	}

	keyCondition = expression.Key(d.index.partitionKey).Equal(expression.Value("cluster"))
	expr, err = expression.NewBuilder().WithKeyCondition(keyCondition).WithFilter(filter).Build()

	if err != nil {
		msg := fmt.Sprintf("Building dynamodb query expersion failed: '%v'.", err)
		log.Errorf(msg)
		return nil, 0, false, fmt.Errorf(msg)
	}

	queryInput = &dynamodb.QueryInput{
		TableName:                 &d.table.name,
		IndexName:                 &d.index.name,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
	}

	for {
		start := time.Now()
		result, err := d.dbAPI.Query(queryInput)
		elapsed := float64(time.Since(start)) / float64(time.Second)

		d.metrics.RecordEgressRequestCnt(egressTarget)
		d.metrics.RecordEgressRequestDur(egressTarget, elapsed)

		if err != nil {
			msg := fmt.Sprintf("DynamonDB API query call failed: '%v'.", err.Error())
			log.Errorf(msg)
			return nil, 0, false, fmt.Errorf(msg)
		}

		for _, i := range result.Items {
			var item ClusterDb

			err = dynamodbattribute.UnmarshalMap(i, &item)
			if err != nil {
				msg := fmt.Sprintf("Error when trying to unmarshal cluster: '%v'.", err.Error())
				log.Errorf(msg)
				return nil, 0, false, fmt.Errorf(msg)
			}
			clusters = append(clusters, *item.Cluster)
		}
		if result.LastEvaluatedKey == nil {
			break
		}
		queryInput.ExclusiveStartKey = result.LastEvaluatedKey
	}

	count := len(clusters)
	startIndex := offset
	endIndex := offset + limit
	more := false

	if endIndex > count {
		endIndex = count
	}
	if endIndex < count {
		more = true
	}

	return clusters[startIndex:endIndex], endIndex - startIndex, more, err
}

func (d *db) ListClustersWithFilter(offset int, limit int, filter *DynamoDBFilter) ([]registryv1.Cluster, int, bool, error) {
	var clusters []registryv1.Cluster = []registryv1.Cluster{}
	var scanInput *dynamodb.ScanInput
	var expr expression.Expression
	var err error

	f, err := filter.Build()
	if err != nil {
		return nil, 0, false, err
	}
	expr, err = expression.NewBuilder().WithFilter(f).Build()

	if err != nil {
		msg := fmt.Sprintf("Building dynamodb scan expersion failed: '%v'.", err)
		log.Errorf(msg)
		return nil, 0, false, fmt.Errorf(msg)
	}

	scanInput = &dynamodb.ScanInput{
		IndexName:                 &d.index.name,
		TableName:                 &d.table.name,
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
	}

	for {
		start := time.Now()
		result, err := d.dbAPI.Scan(scanInput)
		elapsed := float64(time.Since(start)) / float64(time.Second)

		d.metrics.RecordEgressRequestCnt(egressTarget)
		d.metrics.RecordEgressRequestDur(egressTarget, elapsed)

		if err != nil {
			msg := fmt.Sprintf("DynamoDB API scan call failed: '%v'.", err.Error())
			log.Errorf(msg)
			return nil, 0, false, fmt.Errorf(msg)
		}

		for _, i := range result.Items {
			var item ClusterDb

			err = dynamodbattribute.UnmarshalMap(i, &item)
			if err != nil {
				msg := fmt.Sprintf("Error when trying to unmarshal cluster: '%v'.", err.Error())
				log.Errorf(msg)
				return nil, 0, false, fmt.Errorf(msg)
			}
			clusters = append(clusters, *item.Cluster)
		}
		if result.LastEvaluatedKey == nil {
			break
		}
		scanInput.ExclusiveStartKey = result.LastEvaluatedKey
	}

	count := len(clusters)
	startIndex := offset
	endIndex := offset + limit
	more := false

	if endIndex > count {
		endIndex = count
	}
	if endIndex < count {
		more = true
	}

	return clusters[startIndex:endIndex], endIndex - startIndex, more, err
}

// PutCluster (create/update) a cluster in database
func (d *db) PutCluster(cluster *registryv1.Cluster) error {

	lastUpdated, err := time.Parse(time.RFC3339, cluster.Spec.LastUpdated)
	if err != nil {
		msg := fmt.Sprintf("Error converting lastUpdated parameter to RFC3339 for cluster %s: '%v'.", cluster.Spec.Name, err)
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}

	existingCluster, _ := d.GetCluster(cluster.Spec.Name)
	if existingCluster != nil {
		fmt.Printf("Cluster '%s' found in the database. It will be updated.", cluster.Spec.Name)
		cluster.Spec.RegisteredAt = existingCluster.Spec.RegisteredAt
	}

	clusterDb, err := dynamodbattribute.MarshalMap(ClusterDb{
		TablePartitionKey: cluster.Spec.Name,
		IndexPartitionKey: "cluster",
		Region:            cluster.Spec.Region,
		Environment:       cluster.Spec.Environment,
		Status:            cluster.Spec.Status,
		LastUpdatedUnix:   lastUpdated.Unix(),
		Cluster:           cluster,
	})

	if err != nil {
		msg := fmt.Sprintf("Cannot marshal cluster '%s' into AttributeValue map.", cluster.Spec.Name)
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}

	params := &dynamodb.PutItemInput{
		TableName: &d.table.name,
		Item:      clusterDb,
	}

	start := time.Now()
	_, err = d.dbAPI.PutItem(params)
	elapsed := float64(time.Since(start)) / float64(time.Second)

	d.metrics.RecordEgressRequestCnt(egressTarget)
	d.metrics.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		msg := fmt.Sprintf("Cluster '%s' cannot be updated or created in the database. Error: '%v'", cluster.Spec.Name, err.Error())
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}

	log.Infof("Cluster '%s' updated.", cluster.Spec.Name)

	return nil
}

// DeleteCluster delete a cluster from database
func (d *db) DeleteCluster(name string) error {
	params := &dynamodb.DeleteItemInput{
		TableName: &d.table.name,
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	}

	start := time.Now()
	_, err := d.dbAPI.DeleteItem(params)
	elapsed := float64(time.Since(start)) / float64(time.Second)

	d.metrics.RecordEgressRequestCnt(egressTarget)
	d.metrics.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		msg := fmt.Sprintf("Error while deleting cluster %s from db: %v", name, err.Error())
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}

	log.Infof("Cluster %s deleted.", name)

	return nil
}

// ListClustersWithService gets service metadata for a given serviceId on all clusters
func (d *db) ListClustersWithService(serviceId string, offset int, limit int, environment string, region string, status string, lastUpdated string) ([]registryv1.Cluster, int, bool, error) {
	clusters, _, _, err := d.ListClusters(offset, limit, environment, region, status, lastUpdated)

	var clustersWithService []registryv1.Cluster

	for _, cluster := range clusters {
		var serviceMetadata = registryv1.ServiceMetadata{}
		for service := range cluster.Spec.ServiceMetadata {
			if service == serviceId {
				serviceMetadata[service] = cluster.Spec.ServiceMetadata[service]
				cluster.Spec.ServiceMetadata = serviceMetadata
				clustersWithService = append(clustersWithService, cluster)
			}
		}
	}

	count := len(clustersWithService)
	endIndex := offset + limit
	more := false

	if endIndex > count {
		endIndex = count
	}
	if endIndex < count {
		more = true
	}

	return clustersWithService, count, more, err
}

// ListClustersWithServiceAndFilter gets service metadata for a given serviceId on all clusters with additional filtering options
func (d *db) ListClustersWithServiceAndFilter(serviceId string, offset int, limit int, filter *DynamoDBFilter) ([]registryv1.Cluster, int, bool, error) {
	clusters, _, _, err := d.ListClustersWithFilter(offset, limit, filter)

	if err != nil {
		msg := fmt.Sprintf("Failed to list clusters with filter: '%v'.", err)
		log.Errorf(msg)
		return nil, 0, false, fmt.Errorf(msg)
	}

	var clustersWithService []registryv1.Cluster

	for _, cluster := range clusters {
		var serviceMetadata = registryv1.ServiceMetadata{}
		for service := range cluster.Spec.ServiceMetadata {
			if service == serviceId {
				serviceMetadata[service] = cluster.Spec.ServiceMetadata[service]
				cluster.Spec.ServiceMetadata = serviceMetadata
				clustersWithService = append(clustersWithService, cluster)
			}
		}
	}

	count := len(clustersWithService)
	endIndex := offset + limit
	more := false

	if endIndex > count {
		endIndex = count
	}
	if endIndex < count {
		more = true
	}

	return clustersWithService, count, more, err
}

// GetClusterWithService gets service metadata for a given serviceId on a given cluster
func (d *db) GetClusterWithService(serviceId string, clusterName string) (*registryv1.Cluster, error) {
	cluster, err := d.GetCluster(clusterName)

	if err != nil {
		return nil, err
	}

	if cluster == nil {
		return nil, nil
	}

	var serviceMetadata = registryv1.ServiceMetadata{}
	for service := range cluster.Spec.ServiceMetadata {
		if service == serviceId {
			serviceMetadata[service] = cluster.Spec.ServiceMetadata[service]
			cluster.Spec.ServiceMetadata = serviceMetadata
			return cluster, nil
		}
	}

	return nil, err
}
