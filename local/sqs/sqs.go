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
	"io/ioutil"
	"os"
	"time"

	"github.com/adobe/cluster-registry/pkg/api/sqs"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"github.com/adobe/cluster-registry/pkg/cc/monitoring"
	"gopkg.in/yaml.v2"
)

func main() {
	var clusters []registryv1.Cluster

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	m := monitoring.NewMetrics()
	m.Init(false)

	p := sqs.NewProducer(m)
	input_file := flag.String("input-file", "../db/dummy-data.yaml", "yaml file path")
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
		p.Send(ctx, &cluster)
	}

	fmt.Println("Data successfully added into the queue.")
}
