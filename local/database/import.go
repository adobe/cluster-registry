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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/adobe/cluster-registry/pkg/database"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/apiserver"
	"sigs.k8s.io/yaml"
)

func main() {
	var clusters []registryv1.Cluster

	m := monitoring.NewMetrics("cluster_registry_api_local", false)
	appConfig, err := config.LoadApiConfig()
	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	d := database.NewDb(appConfig, m)

	input_file := flag.String("input-file", "dummy-data.yaml", "yaml file path")
	flag.Parse()

	data, err := os.ReadFile(*input_file)
	if err != nil {
		log.Fatalf("Unable to read input file %s: '%v'", *input_file, err.Error())
	}

	dataJson, err := yaml.YAMLToJSON(data)
	if err != nil {
		log.Fatalf("Unable to unmarshal data: '%v'", err.Error())
	}

	err = json.Unmarshal(dataJson, &clusters)
	if err != nil {
		log.Fatalf("Unable to unmarshal data: '%v'", err.Error())
	}

	for _, cluster := range clusters {
		err = d.PutCluster(&cluster)
		if err != nil {
			log.Fatalf("Failed to add or update the cluster %s: '%v'", cluster.Name, err.Error())
		}
	}

	fmt.Println("Import successfully.")
}
