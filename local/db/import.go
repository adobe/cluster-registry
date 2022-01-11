package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	registryv1 "github.com/adobe/cluster-registry/pkg/cc/api/registry/v1"
	"gopkg.in/yaml.v2"
)

func main() {
	var clusters []registryv1.Cluster
	m := monitoring.NewMetrics("cluster_registry_api_local", nil, false)
	d := database.NewDb(m)

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
