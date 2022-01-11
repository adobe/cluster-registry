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
