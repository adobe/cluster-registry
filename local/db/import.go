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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/utils"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"gopkg.in/yaml.v2"
)

func main() {
	var clusters []registryv1.Cluster

	m := monitoring.NewMetrics("cluster_registry_api_local", nil, false)
	appConfig, err := utils.LoadApiConfig()
	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	d := database.NewDb(appConfig, m)

	input_file := flag.String("input-file", "dummy-data.yaml", "yaml file path")
	flag.Parse()

	data, err := ioutil.ReadFile(*input_file)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	err = yaml.Unmarshal([]byte(data), &clusters)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	for _, cluster := range clusters {
		err = d.PutCluster(&cluster)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
		}
	}

	fmt.Println("Import successfully.")
}
