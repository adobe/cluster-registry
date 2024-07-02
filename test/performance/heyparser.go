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

/*
This tool parses the csv output of the hey benchmarking tool.
An example of this output is in 'data/hey_out_example.csv'.

How to use?
hey $endpoint | go run heyparser.go
go run heyparser.go -f data/hey_out_example.csv
*/

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"github.com/adobe/cluster-registry/test/performance/stats"
)

type heyReportRow struct {
	ResponseTime  float64 // Total time taken for request (in seconds)
	DNSPlusDialup float64 // Time taken to establish the TCP connection (in seconds)
	DNS           float64 // Time taken to do the DNS lookup (in seconds)
	RequestWrite  float64 // Time taken to write full request (in seconds)
	ResponseDelay float64 // Time taken to first byte received (in seconds)
	ResponseRead  float64 // Time taken to read full response (in seconds)
	StatusCode    int     // HTTP status code of the response (e.g. 200)
	Offset        float64 // The time since the start of the benchmark when the request was started. (in seconds)
}

func newReportRow(record []string) heyReportRow {
	responseTime, err := strconv.ParseFloat(record[0], 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing file line %s", err.Error())
		os.Exit(1)
	}

	responseDelay, _ := strconv.ParseFloat(record[4], 64)
	dnsPlusDialup, _ := strconv.ParseFloat(record[1], 64)
	dns, _ := strconv.ParseFloat(record[2], 64)
	requestWrite, _ := strconv.ParseFloat(record[3], 64)
	responseRead, _ := strconv.ParseFloat(record[5], 64)
	statusCode, _ := strconv.Atoi(record[6])
	offset, _ := strconv.ParseFloat(record[7], 64)

	return heyReportRow{
		ResponseTime:  responseTime,
		DNSPlusDialup: dnsPlusDialup,
		DNS:           dns,
		RequestWrite:  requestWrite,
		ResponseDelay: responseDelay,
		ResponseRead:  responseRead,
		StatusCode:    statusCode,
		Offset:        offset,
	}
}

func checkOKResponse(respCodes map[int]int) bool {
	keys := make([]int, len(respCodes))
	i := 0
	for k := range respCodes {
		keys[i] = k
		i++
	}

	for _, code := range keys {
		if code >= 400 && code < 500 {
			return false
		}
	}

	return true
}

func main() {

	var endpoint, csvFilePtr string
	flag.StringVar(&endpoint, "e", "", "The endpoint on witch the test ran")
	flag.StringVar(&csvFilePtr, "f", "", "A csv file that contains an hey output")
	flag.Parse()

	file := os.Stdin
	if csvFilePtr != "" {
		file, _ = os.Open(csvFilePtr)
	}
	defer file.Close()

	r := csv.NewReader(file)
	_, _ = r.Read() // Ignore the headers

	var row heyReportRow
	var totalRequests int                // Total number of requests
	var responseTimes []float64          // List with all the response times
	var benchmarkTotalTime float64       // How much did the benchmark took
	statusCodeCount := make(map[int]int) // The total response codes

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR parsing hey report: %s", err.Error())
			os.Exit(1)
		}

		row = newReportRow(record)
		{
			// Update the summary values

			responseTimes = append(responseTimes, row.ResponseTime)

			if row.Offset > benchmarkTotalTime {
				benchmarkTotalTime = row.Offset + row.ResponseRead
			}

			if _, ok := statusCodeCount[row.StatusCode]; !ok {
				statusCodeCount[row.StatusCode] = 0
			}
			statusCodeCount[row.StatusCode]++

			totalRequests++
		}
	}

	if !checkOKResponse(statusCodeCount) {
		fmt.Fprintf(os.Stderr, "ERROR: Response codes:")
		for key, value := range statusCodeCount {
			fmt.Fprintf(os.Stderr, " %v of request of %vs |", value, key)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	percentile, err := stats.Percentile(responseTimes, 99.9)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR calculating the percentile: %s\n", err.Error())
	}

	mean, err := stats.Mean(responseTimes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR calculating the average: %s\n", err.Error())
	}

	fmt.Fprintf(os.Stdout, "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ Benchmark ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓\n")
	if endpoint != "" {
		fmt.Fprintf(os.Stdout, " Endpoint      : %v \n", endpoint)
		fmt.Fprintf(os.Stdout, "┣━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫\n")
	}
	fmt.Fprintf(os.Stdout, " Total Requests: %v ┃", totalRequests)
	fmt.Fprintf(os.Stdout, " Requests/s: %v ┃", math.RoundToEven(float64(totalRequests)/benchmarkTotalTime))
	fmt.Fprintf(os.Stdout, " Average: %.3f ┃", mean)
	fmt.Fprintf(os.Stdout, " 99.9th: %.3f\n", percentile)
	fmt.Fprintf(os.Stdout, "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛\n")

	if err != nil {
		os.Exit(1)
	}
}
