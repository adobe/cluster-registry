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

package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	cr "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	"github.com/adobe/cluster-registry/test/slt/metrics"
)

// ClusterList ...
type ClusterList struct {
	Items      []*cr.ClusterSpec `json:"items"`
	ItemsCount int               `json:"itemsCount"`
}

// GetRandomTime returns a random time interval as a string
func GetRandomTime(maxTime string, logger echo.Logger) string {
	interval, err := time.ParseDuration(maxTime)
	if err != nil {
		logger.Fatalf("error parsing time interval: %s", err.Error())
	}
	maxSeconds := int(interval.Seconds())
	radomSeconds := rand.Intn(maxSeconds)
	return (time.Second * time.Duration(radomSeconds)).String()
}

// RunFuncInLoop runs a function in a loop.
// f is the function to run,
// timeInterval is the time it waits to run again,
// offSetStart is the time to wait before the loop starts.
func RunFuncInLoop(f func(interface{}), config interface{}, timeInterval string, offSetStart string, logger echo.Logger) {
	interval, err := time.ParseDuration(timeInterval)
	if err != nil {
		logger.Fatalf("error parsing time interval to run %s function in loop: %s", GetFunctionName(f), err)
	}

	if offSetStart != "" {
		offSet, err := time.ParseDuration(offSetStart)
		if err != nil {
			logger.Fatalf("error parsing time interval to run %s function in loop: %s", GetFunctionName(f), err)
		}
		time.Sleep(offSet)
	}

	for {
		f(config)
		time.Sleep(interval)
	}
}

// GetEnv gets env variable with an fallback value, if fallback is empty then env variable
// is mandatory and if missing exit the program
func GetEnv(key, fallback string, logger echo.Logger) string {
	if value, ok := os.LookupEnv(key); ok {
		logger.Debugf("got env variable %s:%s", key, value)
		return value
	}

	if fallback == "" {
		logger.Fatalf("missing environment variable %s", key)
	}

	logger.Debugf("could not find env variable %s, set to fallback value %s", key, fallback)
	return fallback
}

// GetFunctionName returns the name ofa function
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func reqGet(endpoint, bearer string) (*[]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot build http request: %s", err.Error())
	}

	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot make http request: %s", err.Error())
	}

	if resp.StatusCode != 200 {
		message, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("status code %d: could "+
				"not read response body: %s", resp.StatusCode, err.Error())
		}
		return nil, resp.StatusCode, fmt.Errorf("cannot get cluster: Status "+
			"code %d, body:%s", resp.StatusCode, string(message))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("cannot read response body: %s", err.Error())
	}

	return &body, resp.StatusCode, nil
}

// GetCluster gets cluster from CR
func GetCluster(url, clusterName, jwtToken string) (*cr.ClusterSpec, error) {
	var cluster cr.ClusterSpec

	endpoint := fmt.Sprintf("%s/api/v1/clusters/%s", url, clusterName)
	bearer := "Bearer " + jwtToken

	start := time.Now()
	body, respCode, err := reqGet(endpoint, bearer)
	timeTook := float64(time.Since(start).Seconds())
	metrics.EgressReqDuration.WithLabelValues(
		"/api/v1/clusters/[cluster]",
		"GET",
		strconv.Itoa(respCode)).Observe(timeTook)
	if err != nil {
		if respCode == 404 {
			return nil, fmt.Errorf("cluster %s was not found: %s", clusterName, err.Error())
		}
		return nil, err
	}

	err = json.Unmarshal(*body, &cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	return &cluster, nil
}

// GetClusters gets cluster from CR
func GetClusters(url, perPageLimit, pageNr, jwtToken string) (*ClusterList, error) {
	var clusters ClusterList

	endpoint := fmt.Sprintf("%s/api/v1/clusters?offset=%s&limit=%s", url, pageNr, perPageLimit)
	bearer := "Bearer " + jwtToken

	start := time.Now()
	body, respCode, err := reqGet(endpoint, bearer)
	timeTook := float64(time.Since(start).Seconds())
	metrics.EgressReqDuration.WithLabelValues(
		"/api/v1/clusters",
		"GET",
		strconv.Itoa(respCode)).Observe(timeTook)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(*body, &clusters)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %s", err.Error())
	}

	return &clusters, nil
}
