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

package client

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func ReadFile(patchFilePath string) ([]byte, error) {
	patchFile, err := os.Open(patchFilePath)
	if err != nil {
		log.Fatalf("Error opening patch YAML file: %v", err)
	}
	defer patchFile.Close()

	patchData, err := io.ReadAll(patchFile)
	if err != nil {
		log.Fatalf("Error reading patch YAML file: %v", err)
	}
	return patchData, nil
}

func UnmarshalYaml(data []byte, cluster *[]registryv1.Cluster) error {
	err := yaml.Unmarshal(data, cluster)
	if err != nil {
		log.Panicf("Error while trying to unmarshal yaml data: %v", err.Error())
	}

	return err
}

func UnmarshalJSON(data []byte, cluster *[]registryv1.Cluster) error {
	err := json.Unmarshal(data, cluster)
	if err != nil {
		log.Panicf("Error while trying to unmarshal json data: %v", err.Error())
	}

	return err
}

func MarshalJson(patch map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(patch)
	if err != nil {
		log.Panicf("Error while trying to marshal json data: %v", err.Error())
	}

	return jsonData, err
}

// toUnstructured converts a Cluster struct to an unstructured.Unstructured object
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return u, nil
}

// TODO; check if there is an utils func - see if no need
// unstructuredToJSON converts an unstructured.Unstructured object to a JSON string
func unstructuredToJSON(obj *unstructured.Unstructured) ([]byte, error) {
	return obj.MarshalJSON()
}

func InitLogger(logLevel string, logFormat string) {

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.DebugLevel
	}
	logrus.SetLevel(level)

	logFormat = strings.ToLower(logFormat)
	if logFormat == "text" {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
}
