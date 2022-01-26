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

package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
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
	appConfig := utils.LoadApiConfig()
	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/clusters", s.apiPort), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", bearer)
	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(body), &clusters)
	if err != nil {
		log.Fatal(err)
	}

	s.Assert().Equal(3, clusters.ItemsCount)
}

func (s *e2eTestSuite) Test_EndToEnd_CreateCluster() {

	var inputCluster registryv1.Cluster
	var outputCluster registryv1.ClusterSpec

	appConfig := utils.LoadApiConfig()
	input_file := "../testdata/cluster05-prod-useast1.json"
	data, err := ioutil.ReadFile(input_file)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = json.Unmarshal([]byte(data), &inputCluster)
	if err != nil {
		log.Fatal(err.Error())
	}

	config, err := clientcmd.BuildConfigFromFlags("", "../../kubeconfig")
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = clientset.CoreV1().RESTClient().
		Post().
		AbsPath("/apis/registry.ethos.adobe.com/v1/namespaces/cluster-registry/clusters").
		Resource("clusters").
		Body(data).
		DoRaw(context.TODO())

	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Successfully created Cluster %s\n", inputCluster.Spec.Name)

	time.Sleep(20 * time.Second)

	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/clusters/%s", s.apiPort, inputCluster.Spec.Name), nil)
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(body), &outputCluster)

	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	inputCluster.Spec.APIServer.CertificateAuthorityData = base64.StdEncoding.EncodeToString(config.CAData)
	inputCluster.Spec.LastUpdated = outputCluster.LastUpdated
	s.Assert().Equal(inputCluster.Spec, outputCluster)

	s.T().Cleanup(
		func() {
			m := monitoring.NewMetrics("cluster_registry_api_e2e_test", nil, true)

			_, err = clientset.CoreV1().RESTClient().
				Delete().
				AbsPath("/apis/registry.ethos.adobe.com/v1").
				Namespace("cluster-registry").
				Resource("clusters").
				Name(inputCluster.Spec.Name).
				DoRaw(context.TODO())

			if err != nil {
				fmt.Printf("Cannot delete Cluster %s\nErr:\n%s", inputCluster.Spec.Name, err.Error())
			}

			d := database.NewDb(appConfig, m)
			d.DeleteCluster(inputCluster.Spec.Name)
		})
}

//fix patch to k8s api
func (s *e2eTestSuite) TBD_Test_EndToEnd_UpdateCluster() {

	var inputCluster registryv1.Cluster
	var outputCluster registryv1.ClusterSpec

	appConfig := utils.LoadApiConfig()
	input_file := "../testdata/cluster05-prod-useast1-update.json"
	data, err := ioutil.ReadFile(input_file)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = json.Unmarshal([]byte(data), &inputCluster)
	if err != nil {
		log.Fatal(err.Error())
	}

	config, err := clientcmd.BuildConfigFromFlags("", "../../kubeconfig")
	if err != nil {
		log.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
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
		fmt.Println(err.Error())
		log.Fatal(err)
	}
	fmt.Printf("Successfully created Cluster %s\n", inputCluster.ClusterName)

	time.Sleep(10 * time.Second)

	jwtToken := jwt.GenerateDefaultSignedToken(appConfig)
	bearer := "Bearer " + jwtToken
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/api/v1/clusters/%s", s.apiPort, inputCluster.Spec.Name), nil)
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

	body, err := ioutil.ReadAll(resp.Body)
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
