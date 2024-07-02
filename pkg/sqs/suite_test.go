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

package sqs

import (
	"context"
	"fmt"
	"github.com/onsi/gomega/gexec"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	"time"
)

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type sqsContainer struct {
	testcontainers.Container
	Endpoint string
}

var sqsTestConfig = map[string]string{
	"AWS_ACCESS_KEY_ID": "aws-access-key",
	"AWS_SECRET":        "aws-secret-access-key",
	"AWS_REGION":        "aws-region",
	"SQS_QUEUE_NAME":    "cluster-registry-local",
}

var container *sqsContainer

func TestSQS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SQS Suite")
}

var _ = BeforeSuite(func() {
	var err error

	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx := context.Background()

	By("bootstrapping test environment")
	for k, v := range sqsTestConfig {
		_ = os.Setenv(k, v)
	}

	container, err = setupSQS(ctx)
	if err != nil {
		Fail(fmt.Sprintf("Error while creating the SQS container: %v", err))
	}

	log.Log.Info("SQS container running", "endpoint", container.Endpoint)
})

var _ = AfterSuite(func() {
	ctx := context.Background()

	By("tearing down the test environment")
	gexec.KillAndWait(5 * time.Second)

	err := container.Terminate(ctx)
	if err != nil {
		Fail(fmt.Sprintf("Error while terminating the SQS container: %v", err))
	}

	for k := range sqsTestConfig {
		_ = os.Unsetenv(k)
	}
})

func setupSQS(ctx context.Context) (*sqsContainer, error) {
	sqsConfigPath, err := filepath.Abs("../../local/sqs/sqs.conf")
	if err != nil {
		return nil, err

	}
	req := testcontainers.ContainerRequest{
		Name:         "sqs-test",
		Image:        "softwaremill/elasticmq-native:1.5.7",
		ExposedPorts: []string{"9324/tcp"},
		Entrypoint: []string{
			"/sbin/tini",
			"--",
			"/opt/elasticmq/bin/elasticmq-native-server",
			"-Dconfig.file=/opt/elasticmq.conf",
			"-Dlogback.configurationFile=/opt/logback.xml",
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      sqsConfigPath,
				ContainerFilePath: "/opt/elasticmq.conf",
				FileMode:          700,
			},
		},
		WorkingDir: "/opt/elasticmq",
		WaitingFor: wait.ForListeningPort("9324/tcp"),
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

	mappedPort, err := container.MappedPort(ctx, "9324")
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("http://%s:%s", ip, mappedPort.Port())

	return &sqsContainer{Container: container, Endpoint: endpoint}, nil
}
