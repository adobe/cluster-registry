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
This is a service that runs the SLT check and saves the metrics for prometheus.
*/

package main

import (
	"time"

	"github.com/labstack/gommon/log"
	"github.com/prometheus/client_golang/prometheus"

	slt "github.com/adobe/cluster-registry/test/slt/slt"
	web "github.com/adobe/cluster-registry/test/slt/web"
)

var logger *log.Logger
var timeBetweenSLTs time.Duration // Minutes to wait between SLTs

var sltStatus = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "slt_state",
		Help: "The status of the last SLT, has values between 1 if the check passed " +
			"and 0 if it failed.",
	},
)

func runSLTLoop() {
	// Wait a sec for the server to start
	time.Sleep(1 * time.Second)

	slt.AddConfig(slt.GetConfigFromEnv())

	for true {
		status, err := slt.Run()
		if err != nil {
			logger.Fatal(err)
		}
		sltStatus.Set(status)

		// The time between SLTs
		time.Sleep(timeBetweenSLTs)
	}
}

// Runs before main
func init() {
	logger = log.New("echo")
	slt.SetLogger(logger)

	aux, err := time.ParseDuration(slt.GetEnv("TIME_BETWEEN_SLT", "1m"))
	if err != nil {
		logger.Fatalf("Error converting `TIME_BETWEEN_SLT` value: %s", err)
	}
	timeBetweenSLTs = aux

	prometheus.Register(sltStatus)
}

func main() {
	e := web.NewEchoWithLogger(logger)

	e.GET("/metrics", web.Metrics())
	e.GET("/livez", web.Livez)

	go runSLTLoop()

	e.Logger.Fatal(e.Start(":8081"))
}
