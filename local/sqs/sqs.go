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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/config"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/adobe/cluster-registry/pkg/sqs"
	"gopkg.in/yaml.v2"
)

func main() {
	var clusters []registryv1.Cluster

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m := monitoring.NewMetrics()
	m.Init(false)
	appConfig, err := config.LoadApiConfig()
	if err != nil {
		log.Fatalf("Cannot load the api configuration: '%v'", err.Error())
	}

	p := sqs.NewProducer(appConfig, m)
	input_file := flag.String("input-file", "../db/dummy-data.yaml", "yaml file path")
	flag.Parse()

	data, err := os.ReadFile(*input_file)
	if err != nil {
		log.Panicf("Error while trying to read file: %v", err.Error())
	}

	err = yaml.Unmarshal([]byte(data), &clusters)
	if err != nil {
		log.Panicf("Error while trying to unmarshal data: %v", err.Error())
	}

	for _, cluster := range clusters {
		err = p.Send(ctx, &cluster)
		if err != nil {
			log.Panicf("Error sending message to sqs: %v", err.Error())
		}
	}

	fmt.Println("Data successfully added into the queue.")
}
