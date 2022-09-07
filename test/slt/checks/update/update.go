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
This is a e2e test that checks if the cluster registry client reacts after an CRD update
and pushes the changes to the APIs database.
*/

package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	h "github.com/adobe/cluster-registry/test/slt/helpers"

	cr "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/labstack/echo/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// The key name of the tag to update for the slt test
const tagSLT = "update-slt"

var logger echo.Logger

// TestConfig holds the config for this test
type TestConfig struct {
	url       string
	namespace string
}

// SetLogger sets the global logger for the slt package
func SetLogger(lgr echo.Logger) {
	logger = lgr
}

// GetConfigFromEnv gets from the env the needed global env
func GetConfigFromEnv() TestConfig {
	return TestConfig{
		url:       h.GetEnv("URL", "http://localhost:8080", logger),
		namespace: h.GetEnv("NAMESPACE", "cluster-registry", logger),
	}
}

func updateCrd(namespace string) (string, string, error) {
	var clusterList cr.ClusterList

	cfg, err := config.GetConfig()
	if err != nil {
		return "", "", fmt.Errorf("could not create Kubernetes config: %s", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", "", fmt.Errorf("could not create Kubernetes client: %s", err.Error())
	}

	result, err := clientset.CoreV1().RESTClient().
		Get().
		AbsPath("/apis/registry.ethos.adobe.com/v1").
		Resource("clusters").
		Namespace(namespace).Do(context.TODO()).Raw()
	if err != nil {
		return "", "", fmt.Errorf("could not get the custom resource: %s. Response %s",
			err.Error(), string(result))
	}

	err = json.Unmarshal(result, &clusterList)
	if err != nil {
		return "", "", fmt.Errorf("could not unmarshal Kubernetes response: %s", err.Error())
	}

	if len(clusterList.Items) == 0 {
		return "", "", errors.New("no CRD found, the 'Items' list is empty")
	}

	// There should only be one CRD in the cluster
	cluster := &clusterList.Items[0]

	if (*cluster).Spec.Tags == nil {
		logger.Infof("creating '%s' tag with the 'Tick' value...", tagSLT)
		(*cluster).Spec.Tags = map[string]string{tagSLT: "Tick"}
	} else if value, ok := (*cluster).Spec.Tags[tagSLT]; ok {
		if value == "Tick" {
			logger.Infof("changing '%s' tag value from '%s' to '%s'...",
				tagSLT, value, "Tack")
			(*cluster).Spec.Tags[tagSLT] = "Tack"
		} else if value == "Tack" {
			logger.Infof("changing '%s' tag value from '%s' to '%s'...",
				tagSLT, value, "Tick")
			(*cluster).Spec.Tags[tagSLT] = "Tick"
		} else {
			return "", "", fmt.Errorf("found '%s' tag with wrong value '%s'", tagSLT, value)
		}
	} else {
		logger.Infof("creating '%s' tag with the 'Tick' value...", tagSLT)
		(*cluster).Spec.Tags[tagSLT] = "Tick"
	}

	// Remove immutable Kubernetes field
	(*cluster).ObjectMeta.ManagedFields = []metav1.ManagedFieldsEntry{}

	data, err := json.Marshal(*cluster)
	if err != nil {
		return "", "", fmt.Errorf("could not marshal updated CRD: %s", err.Error())
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
		return "", "", fmt.Errorf("could not update the CRD: %s", err.Error())
	}

	return (*cluster).Spec.Name, (*cluster).Spec.Tags[tagSLT], nil
}

func checkAPIforUpdate(url, clusterName, tagSLTValue, jwtToken string) error {
	cluster, err := h.GetCluster(url, clusterName, jwtToken)
	if err != nil {
		return err
	}

	if cluster.Tags == nil {
		return errors.New("tags field is empty")
	} else if tagSLTValue != cluster.Tags[tagSLT] {
		return fmt.Errorf("the 'Tags' field is not what expected. The "+
			"value is '%s', expected '%s'.", cluster.Tags[tagSLT], tagSLTValue)
	}

	return nil
}

// Run runs the SLT
func Run(config TestConfig, jwtToken string) (int, int, error) {
	logger.Info("updating the Cluster Registry CRD...")
	clusterName, tagSLTValue, err := updateCrd(config.namespace)
	if err != nil {
		return 0, 0, err
	}
	logger.Info("Cluster Registry CRD updated!")

	logger.Info("waiting for the Cluster Registry API to update the database...")
	maxNrOfTries, nrOfTries := 3, 1
	for nrOfTries <= maxNrOfTries {
		// Give to the CR client time to push to the SQS queue and for the API to read
		// from the queue and update the DB. By local tests it takes around 11s
		time.Sleep(11 * time.Second)

		logger.Infof("checking the API for the update (check %d/%d)...",
			nrOfTries, maxNrOfTries)

		err = checkAPIforUpdate(config.url, clusterName, tagSLTValue, jwtToken)
		if err != nil {
			logger.Error(err.Error())
			nrOfTries++
			continue
		}

		logger.Info("update confirmed")
		return 1, nrOfTries, nil
	}

	logger.Error("failed to confirm the update")
	return 0, 0, nil
}
