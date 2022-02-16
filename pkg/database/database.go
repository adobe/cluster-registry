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

// Db provides an interface for interacting with dynamonDb
type Db interface {
	GetCluster(name string) (*registryv1.Cluster, error)
	ListClusters(offset int, limit int, environment string, region string, status string) ([]registryv1.Cluster, int, bool, error)
	PutCluster(cluster *registryv1.Cluster) error
	DeleteCluster(name string) error
	Status() error
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
		log.Errorf("Connectivity check using DescribeTable failed. Error: '%v'", err.Error())
		return err
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
		log.Errorf("Cannot get cluster '%s' from the database. Error: '%v'", name, err.Error())
		return nil, err
	}

	if resp.Item == nil {
		log.Warnf("Cluster '%s' not found in the database.", name)
		return nil, nil
	}

	var clusterDb *ClusterDb
	err = dynamodbattribute.UnmarshalMap(resp.Item, &clusterDb)
	if err != nil {
		log.Errorf("Cannot unmarshal cluster '%s'. Error: '%v'", err.Error())
		return nil, err
	}

	return clusterDb.Cluster, err
}

// ListClusters list all clusters
func (d *db) ListClusters(offset int, limit int, region string, environment string, status string) ([]registryv1.Cluster, int, bool, error) {

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

	keyCondition = expression.Key(d.index.partitionKey).Equal(expression.Value("cluster"))
	expr, err = expression.NewBuilder().WithKeyCondition(keyCondition).WithFilter(filter).Build()

	if err != nil {
		log.Errorf("Building dynamodb query expersion failed: '%v'.", err.Error())
		return nil, 0, false, err
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
			log.Errorf("DynamonDB API query call failed: '%v'.", err.Error())
		}

		for _, i := range result.Items {
			var item ClusterDb

			err = dynamodbattribute.UnmarshalMap(i, &item)
			if err != nil {
				log.Errorf("Got error when trying to unmarshal cluster: '%v'.", err.Error())
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

// PutCluster (create/update) a cluster in database
func (d *db) PutCluster(cluster *registryv1.Cluster) error {

	clusterDb, err := dynamodbattribute.MarshalMap(ClusterDb{
		TablePartitionKey: cluster.Spec.Name,
		IndexPartitionKey: "cluster",
		Region:            cluster.Spec.Region,
		Environment:       cluster.Spec.Environment,
		Status:            cluster.Spec.Status,
		Cluster:           cluster,
	})

	if err != nil {
		log.Errorf("Cannot marshal cluster '%s' into AttributeValue map.", cluster.Name)
		return err
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
		log.Errorf("Cluster '%s' cannot be updated or created in the database. Error: '%v'", cluster.Name, err.Error())
		return err
	}

	log.Infof("Cluster '%s' updated successfully.", cluster.Name)

	return err
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
		log.Errorf("Cluster deletion error from db: %v", err.Error())
		return err
	}

	log.Infof("Cluster %s deleted successfully", name)

	return err
}
