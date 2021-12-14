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
	"os"
	"reflect"
	"time"

	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
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
	ListClusters(region string, environment string, businessUnit string, status string) ([]registryv1.Cluster, int, error)
	PutCluster(cluster *registryv1.Cluster) error
	DeleteCluster(name string) error
}

// db struct
type db struct {
	dbAPI     dynamodbiface.DynamoDBAPI
	tableName string
	met       monitoring.MetricsI
}

// ClusterDb encapsulates the Cluster CRD
type ClusterDb struct {
	Name    string              `json:"name,hash"`
	Cluster *registryv1.Cluster `json:"crd,hash"`
}

// NewDb func
func NewDb(m monitoring.MetricsI) Db {
	dbEndpoint := os.Getenv("DB_ENDPOINT")
	awsRegion := os.Getenv("DB_AWS_REGION")
	dbTableName := os.Getenv("DB_TABLE_NAME")

	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String(awsRegion),
		Endpoint: aws.String(dbEndpoint),
	}))

	d := dynamodb.New(sess)
	dbInst := &db{
		dbAPI:     d,
		tableName: dbTableName,
		met:       m,
	}

	return dbInst
}

// GetCluster a single cluster
func (d *db) GetCluster(name string) (*registryv1.Cluster, error) {
	params := &dynamodb.GetItemInput{
		TableName: &d.tableName,
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	}

	start := time.Now()
	resp, err := d.dbAPI.GetItem(params)
	elapsed := float64(time.Since(start)) / float64(time.Second)

	d.met.RecordEgressRequestCnt(egressTarget)
	d.met.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		log.Warn(err.Error())
		return nil, err
	}

	if resp.Item == nil {
		log.Warn("Cluster " + name + " not found")
		return nil, nil
	}

	var clusterDb *ClusterDb
	err = dynamodbattribute.UnmarshalMap(resp.Item, &clusterDb)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return clusterDb.Cluster, err
}

// ListClusters all clusters
func (d *db) ListClusters(region string, environment string, businessUnit string, status string) ([]registryv1.Cluster, int, error) {
	var clusters []registryv1.Cluster = []registryv1.Cluster{}
	var err error

	// add all params to a map
	queryParams := make(map[string]string, 4)
	var params *dynamodb.ScanInput
	var filt expression.ConditionBuilder

	if region != "" {
		queryParams["region"] = region
	}
	if environment != "" {
		queryParams["environment"] = environment
	}
	if businessUnit != "" {
		queryParams["businessUnit"] = businessUnit
	}
	if status != "" {
		queryParams["status"] = status
	}

	// TODO: use nested filtering
	if len(queryParams) > 0 {
		for k, v := range queryParams {
			if reflect.DeepEqual(filt, expression.ConditionBuilder{}) {
				filt = expression.Name(k).Equal(expression.Value(v))
			} else {
				filt = filt.And(expression.Name(k).Equal(expression.Value(v)))
			}
		}

		expr, err := expression.NewBuilder().WithFilter(filt).Build()
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		params = &dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			TableName:                 aws.String(d.tableName),
		}
	} else {
		params = &dynamodb.ScanInput{
			TableName: aws.String(d.tableName),
		}
	}

	for {
		start := time.Now()
		result, err := d.dbAPI.Scan(params)
		elapsed := float64(time.Since(start)) / float64(time.Second)

		d.met.RecordEgressRequestCnt(egressTarget)
		d.met.RecordEgressRequestDur(egressTarget, elapsed)

		if err != nil {
			log.Error("Query DynamonDB API call failed: " + err.Error())
		}

		for _, i := range result.Items {
			item := ClusterDb{}

			err = dynamodbattribute.UnmarshalMap(i, &item)

			if err != nil {
				log.Error("Got error unmarshalling: " + err.Error())
			}

			clusters = append(clusters, *item.Cluster)
		}
		if result.LastEvaluatedKey == nil {
			break
		}
		params.ExclusiveStartKey = result.LastEvaluatedKey
	}

	return clusters, len(clusters), err
}

// PutCluster (create/update) a cluster in database
func (d *db) PutCluster(cluster *registryv1.Cluster) error {

	clusterDb, err := dynamodbattribute.MarshalMap(ClusterDb{Name: cluster.Spec.Name, Cluster: cluster})
	if err != nil {
		log.Error("Cannot marshal cluster into AttributeValue map")
		return err
	}

	params := &dynamodb.PutItemInput{
		TableName: &d.tableName,
		Item:      clusterDb,
	}

	start := time.Now()
	_, err = d.dbAPI.PutItem(params)
	elapsed := float64(time.Since(start)) / float64(time.Second)

	d.met.RecordEgressRequestCnt(egressTarget)
	d.met.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("Cluster " + cluster.Name + " updated successfully")

	return err
}

// DeleteCluster delete a cluster from database
func (d *db) DeleteCluster(name string) error {
	params := &dynamodb.DeleteItemInput{
		TableName: &d.tableName,
		Key: map[string]*dynamodb.AttributeValue{
			"name": {
				S: aws.String(name),
			},
		},
	}

	start := time.Now()
	_, err := d.dbAPI.DeleteItem(params)
	elapsed := float64(time.Since(start)) / float64(time.Second)

	d.met.RecordEgressRequestCnt(egressTarget)
	d.met.RecordEgressRequestDur(egressTarget, elapsed)

	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("Cluster " + name + " deleted successfully")

	return err
}

// DeleteAllClusters
