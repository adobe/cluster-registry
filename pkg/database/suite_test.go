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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
)

var dbContainer *datatbaseContainer
var dbTestConfig = map[string]string{
	"AWS_ACCESS_KEY_ID":     "aws-access-key",
	"AWS_SECRET_ACCESS_KEY": "aws-secret-access-key",
	"DB_TABLE_NAME":         "cluster-registry-local",
	"DB_INDEX_NAME":         "search-index-local",
	"AWS_REGION":            "aws-region",
}

type datatbaseContainer struct {
	testcontainers.Container
	Endpoint string
}

func TestDatabase(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Database Suite")
}

var _ = BeforeSuite(func() {
	var err error
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx := context.Background()

	By("bootstrapping test environment")

	// needed for aws cli
	for k, v := range dbTestConfig {
		os.Setenv(k, v)
	}

	dbContainer, err = setupDatabse(ctx)
	if err != nil {
		log.Fatalf("Error while creating the database container: %v", err.Error())
	}
	log.Printf("Database container is running at endpoint: %s", dbContainer.Endpoint)
})

var _ = AfterSuite(func() {
	ctx := context.Background()

	By("tearing down the test environment")
	gexec.KillAndWait(5 * time.Second)

	err := dbContainer.Terminate(ctx)
	if err != nil {
		log.Fatalf("Error while creating the database container: %v", err.Error())
	}

	for k := range dbTestConfig {
		os.Unsetenv(k)
	}
})

func setupDatabse(ctx context.Context) (*datatbaseContainer, error) {
	req := testcontainers.ContainerRequest{
		Name:         "database",
		Image:        "amazon/dynamodb-local:1.16.0",
		ExposedPorts: []string{"8000/tcp"},
		Cmd:          []string{"-jar", "DynamoDBLocal.jar", "-inMemory", "-sharedDb"},
		WaitingFor:   wait.ForListeningPort("8000/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "8000")
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	return &datatbaseContainer{Container: container, Endpoint: endpoint}, nil
}

func createTable(endpoint string) error {
	log.Println("Create new database table")
	cmd := exec.Command("aws", "dynamodb", "create-table", "--cli-input-json", "file://testdata/schema.json", "--endpoint-url", endpoint)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func deleteTable(endpoint string, tableName string) error {
	log.Printf("Create database table %s\n", tableName)
	cmd := exec.Command("aws", "dynamodb", "delete-table", "--endpoint-url", endpoint, "--table-name", tableName)
	_, err := cmd.Output()
	if err != nil {
		return err
	}
	return nil
}

func importData(db Db) error {
	var clusters []registryv1.Cluster

	data, err := ioutil.ReadFile("testdata/clusters.yaml")
	if err != nil {
		return err
	}

	dataJson, err := yaml.YAMLToJSON(data)
	if err != nil {
		return err
	}

	err = json.Unmarshal(dataJson, &clusters)
	if err != nil {
		return err
	}

	log.Printf("Populating database with dummy data.")

	for _, cluster := range clusters {
		err = db.PutCluster(&cluster)
		if err != nil {
			return fmt.Errorf("Failed to add or update the cluster %s: '%v'", cluster.Spec.Name, err.Error())
		}
	}
	return nil
}
