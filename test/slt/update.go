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

/*
This is an SLT that checks if the cluster registry client reacts after an CRD update
and pushes the changes to the APIs database.
*/

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const tagSLT = "update-slt"

// This vars will get overwritten by env vars if they exists
var url, namespace string

var jwtToken string

// Gets env variable with an fallback value, if fallback is empty then env variable
// is mandatory and if missing exit the program
func getEnv(key, fallback string) string {

	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	if fallback == "" {
		log.Printf("Missing environment variable %s", key)
		os.Exit(1)
	}
	return fallback
}

func readTokenFromFile(filePath string) (string, error) {

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("Error reading token from file %s: %s", filePath, err.Error())
	}

	return string(data), nil
}

func updateCrd() (string, string, error) {

	var clusterList registryv1.ClusterList

	cfg, err := config.GetConfig()
	if err != nil {
		return "", "", fmt.Errorf("Could not create Kubernetes config: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", "", fmt.Errorf("Could not create Kubernetes client: %s", err.Error())
	}

	result, err := clientset.CoreV1().RESTClient().
		Get().
		AbsPath("/apis/registry.ethos.adobe.com/v1").
		Resource("clusters").
		Namespace(namespace).Do(context.TODO()).Raw()
	if err != nil {
		return "", "", fmt.Errorf("Could not get the custom resource: %s. Response %s",
			err.Error(), string(result))
	}

	err = json.Unmarshal(result, &clusterList)
	if err != nil {
		return "", "", fmt.Errorf("Could not unmarshal Kubernetes response: %s", err.Error())
	}

	if len(clusterList.Items) == 0 {
		return "", "", errors.New("No CRD found, the 'Items' list is empty")
	}

	// There should only be one CRD in the cluster
	cluster := &clusterList.Items[0]

	if (*cluster).Spec.Tags == nil {
		log.Printf("Creating '%s' tag with the 'Tick' value...", tagSLT)
		(*cluster).Spec.Tags = map[string]string{tagSLT: "Tick"}

	} else if (*cluster).Spec.Tags[tagSLT] == "Tick" {
		log.Printf("Changing '%s' tag value from '%s' to '%s'...",
			tagSLT, (*cluster).Spec.Tags[tagSLT], "Tack")
		(*cluster).Spec.Tags[tagSLT] = "Tack"

	} else if (*cluster).Spec.Tags[tagSLT] == "Tack" {
		log.Printf("Changing '%s' tag value from '%s' to '%s'...",
			tagSLT, (*cluster).Spec.Tags[tagSLT], "Tick")
		(*cluster).Spec.Tags[tagSLT] = "Tick"
	}

	// Remove immutable Kubernetes field
	(*cluster).ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}

	data, err := json.Marshal(*cluster)
	if err != nil {
		return "", "", fmt.Errorf("Could not marshal updated CRD: %s", err.Error())
	}

	_, err = clientset.CoreV1().RESTClient().
		Patch(types.MergePatchType).
		AbsPath("/apis/registry.ethos.adobe.com/v1").
		Resource("clusters").
		Namespace(namespace).
		Name((*cluster).Spec.Name).
		Body(data).
		Do(context.TODO()).Raw()
	if err != nil {
		return "", "", fmt.Errorf("Could not update the CRD: %s", err.Error())
	}

	return (*cluster).Spec.Name, (*cluster).Spec.Tags[tagSLT], nil
}

func checkAPIforUpdate(clusterName, tagSLTValue string) error {

	var cluster registryv1.ClusterSpec

	endpoint := fmt.Sprintf("%s/api/v1/clusters/%s", url, clusterName)
	bearer := "Bearer " + jwtToken

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("Cannot build http request: %s", err.Error())
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Cannot make http request: %s", err.Error())
	}

	if resp.StatusCode != 200 {
		message, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Cannot get cluster. Status code %d. Could"+
				"not read response body: %s", resp.StatusCode, err.Error())
		}
		return fmt.Errorf("Cannot get cluster: Status code %d, body:%s",
			resp.StatusCode, string(message))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Cannot read response body: %s", err.Error())
	}

	err = json.Unmarshal([]byte(body), &cluster)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal response: %s", err.Error())
	}

	if cluster.Tags == nil {
		return errors.New("Tags field is empty")
	} else if tagSLTValue != cluster.Tags[tagSLT] {
		return fmt.Errorf("The 'Tags' field is not what expected. The "+
			"value is '%s', expected '%s'.", cluster.Tags[tagSLT], tagSLTValue)
	}

	return nil

}

func main() {

	tokenPath := getEnv("TOKEN_PATH", "")
	url = getEnv("URL", "http://localhost:8080")
	namespace = getEnv("NAMESPACE", "cluster-registry")

	log.Printf("Reading the Cluster Registry API token from '%s'", tokenPath)
	data, err := readTokenFromFile(tokenPath)
	jwtToken = data
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	log.Println("Updating the Cluster Registry CRD...")
	clusterName, tagSLTValue, err := updateCrd()
	if err != nil {
		log.Printf("[ERROR] %s", err.Error())
		os.Exit(1)
	}
	log.Println("Cluster Registry CRD updated!")

	log.Println("Waiting for the Cluster Registry API to update the database...")
	maxNrOfTries, nrOfTries := 3, 1
	updateConfirmed := false
	for nrOfTries <= maxNrOfTries {
		// Give to the CR client time to push to the SQS queue and for the API to read
		// from the queue and update the DB. By local tests it takes around 11s
		time.Sleep(11 * time.Second)

		log.Println(fmt.Sprintf("Checking the API for the update (check %d/%d)...",
			nrOfTries, maxNrOfTries))
		nrOfTries++

		err = checkAPIforUpdate(clusterName, tagSLTValue)
		if err != nil {
			log.Printf("[ERROR] %s", err.Error())
			continue
		}

		log.Println("Update confirmed!")
		updateConfirmed = true
		break
	}

	if !updateConfirmed {
		log.Println("[ERROR] Failed to confirm the update!")
		os.Exit(1)
	}

}
