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

package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"github.com/adobe/cluster-registry/test/jwt"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type e2eTestSuite struct {
	suite.Suite
	apiPort int
}

type clusterList struct {
	Items      []*registryv1.ClusterSpec `json:"items"`
	ItemsCount int                       `json:"itemsCount"`
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, &e2eTestSuite{})
}

func (s *e2eTestSuite) SetupSuite() {
	s.apiPort = 8080
}

func (s *e2eTestSuite) TearDownSuite() {
}

func (s *e2eTestSuite) SetupTest() {
}

func (s *e2eTestSuite) TearDownTest() {

}

func (s *e2eTestSuite) Test_EndToEnd_GetClusters() {

	var clusters clusterList
	appConfig, err := config.LoadApiConfig()
	if err != nil {
		s.T().Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/clusters", s.apiPort), nil)
	if err != nil {
		s.T().Fatalf("Cannot build http request: %v", err.Error())
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.T().Fatalf("Cannot make http request: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		s.T().Fatalf("Cannot list clusters: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.T().Fatalf("Cannot read response body: %v", err.Error())
	}

	err = json.Unmarshal([]byte(body), &clusters)
	if err != nil {
		s.T().Fatalf("Failed to unmarshal data: %v", err.Error())
	}

	s.Assert().Equal(3, clusters.ItemsCount)
}

func (s *e2eTestSuite) Test_EndToEnd_CreateCluster() {

	var inputCluster registryv1.Cluster
	var outputCluster registryv1.ClusterSpec

	input_file := "../testdata/cluster05-prod-useast1.json"
	data, err := os.ReadFile(input_file)
	if err != nil {
		s.T().Fatalf("Failed to read data from file %s.", input_file)
	}

	err = json.Unmarshal([]byte(data), &inputCluster)
	if err != nil {
		s.T().Fatalf("Failed to unmarshal data: %v", err.Error())
	}

	clientConfig, err := clientcmd.BuildConfigFromFlags("", "../../kubeconfig")
	if err != nil {
		s.T().Fatalf("Failed to build K8s config: %v", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		s.T().Fatalf("Failed to build K8s clientset: %v", err.Error())
	}

	_, err = clientset.CoreV1().RESTClient().
		Post().
		AbsPath("/apis/registry.ethos.adobe.com/v1/namespaces/cluster-registry/clusters").
		Resource("clusters").
		Body(data).
		DoRaw(context.TODO())

	if err != nil {
		s.T().Fatalf("Failed to create object %s on the k8s api: %v", inputCluster.Name, err.Error())
	}
	s.T().Logf("Successfully created cluster %s.", inputCluster.Spec.Name)

	time.Sleep(20 * time.Second)

	appConfig, err := config.LoadApiConfig()
	if err != nil {
		s.T().Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v2/clusters/%s", s.apiPort, inputCluster.Spec.Name), nil)

	if err != nil {
		s.T().Fatalf("Failed to build request object: %v", err.Error())
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		s.T().Fatalf("Failed to query clusters from cluster-registry-api: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		s.T().Fatalf("Cannot list clusters: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.T().Fatalf("Failed read response body: %v", err.Error())
	}

	err = json.Unmarshal([]byte(body), &outputCluster)

	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	inputCluster.Spec.APIServer.CertificateAuthorityData = base64.StdEncoding.EncodeToString(clientConfig.CAData)
	inputCluster.Spec.LastUpdated = outputCluster.LastUpdated
	s.Assert().Equal(inputCluster.Spec, outputCluster)

	s.T().Cleanup(
		func() {
			m := monitoring.NewMetrics("cluster_registry_api_e2e_test", true)

			_, err = clientset.CoreV1().RESTClient().
				Delete().
				AbsPath("/apis/registry.ethos.adobe.com/v1").
				Namespace("cluster-registry").
				Resource("clusters").
				Name(inputCluster.Spec.Name).
				DoRaw(context.TODO())

			if err != nil {
				s.T().Fatalf("Cannot delete cluster %s: %v", inputCluster.Spec.Name, err.Error())
			}
			s.T().Logf("Successfully delete cluster %s from k8s api.", inputCluster.Spec.Name)

			d := database.NewDb(appConfig, m)
			err := d.DeleteCluster(inputCluster.Spec.Name)
			if err != nil {
				s.T().Fatalf("Error wihle trying to delete the cluster from database: %v", err.Error())
			}
			s.T().Logf("Successfully delete cluster %s from database.", inputCluster.Spec.Name)
		})
}

// fix patch to k8s api
func (s *e2eTestSuite) TBD_Test_EndToEnd_UpdateCluster() {

	var inputCluster registryv1.Cluster
	var outputCluster registryv1.ClusterSpec

	appConfig, err := config.LoadApiConfig()
	if err != nil {
		s.T().Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	input_file := "../testdata/cluster05-prod-useast1-update.json"
	data, err := os.ReadFile(input_file)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = json.Unmarshal([]byte(data), &inputCluster)
	if err != nil {
		log.Fatal(err.Error())
	}

	clientConfig, err := clientcmd.BuildConfigFromFlags("", "../../kubeconfig")
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = clientset.CoreV1().RESTClient().
		Patch(types.JSONPatchType).
		AbsPath("/apis/registry.ethos.adobe.com/v1").
		Resource("clusters").
		Namespace("cluster-registry").
		Name(inputCluster.Spec.Name).
		Body(data).
		DoRaw(context.TODO())

	if err != nil {
		s.T().Fatalf("Falied to create object %s into k8s api.", inputCluster.Spec.Name)
	}

	s.T().Logf("Successfully created cluster %s.", inputCluster.Spec.Name)

	time.Sleep(10 * time.Second)

	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v2/clusters/%s", s.apiPort, inputCluster.Spec.Name), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(body), &outputCluster)
	if err != nil {
		log.Fatal(err)
	}

	s.Assert().Equal(resp.StatusCode, http.StatusOK)
	s.Assert().Equal(inputCluster.Spec.Status, outputCluster.Status)
}

func (s *e2eTestSuite) Test_EndToEnd_RateLimiter() {

	appConfig, err := config.LoadApiConfig()
	if err != nil {
		s.T().Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v2/clusters", s.apiPort), nil)
	if err != nil {
		s.T().Fatalf("Cannot build http request: %v", err.Error())
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}

	statusOK := 0
	statusTooManyRequests := 0
	requests_nr := 200
	expectedMaxStatusOK := 150

	for i := 0; i < requests_nr; i++ {
		resp, err := client.Do(req)
		if err != nil {
			s.T().Fatalf("Cannot make http request: %v", err.Error())
		}
		defer resp.Body.Close()

		s.NoError(err)

		if resp.StatusCode == http.StatusOK {
			statusOK += 1
		} else if resp.StatusCode == http.StatusTooManyRequests {
			statusTooManyRequests += 1
		} else {
			s.T().Errorf("Unexpected status code: %d", resp.StatusCode)
		}
	}

	s.Assert().LessOrEqual(statusOK, expectedMaxStatusOK)
}
