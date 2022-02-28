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

package stats

import (
	"errors"
	"math"
	"sort"
)

// Sum a
func Sum(input []float64) (sum float64, err error) {
	if len(input) == 0 {
		return math.NaN(), errors.New("The array given is empty")
	}

	for _, n := range input {
		sum += n
	}

	return sum, nil
}

//Median returns the median
func Median(input []float64) (median float64, err error) {

	// Start by sorting a copy of the slice
	c := sortedCopy(input)
	l := len(c)
	if l == 0 {
		return math.NaN(), errors.New("The array given is empty")
	} else if l%2 == 0 {
		median, _ = Mean(c[l/2-1 : l/2+1])
	} else {
		median = c[l/2]
	}

	return median, nil
}

func sortedCopy(input []float64) []float64 {
	c := make([]float64, len(input))
	copy(c, input)
	sort.Float64s(c)

	return c
}

// Mean returns the mean
func Mean(input []float64) (float64, error) {
	if len(input) == 0 {
		return math.NaN(), errors.New("The array given is empty")
	}

	sum, _ := Sum(input)

	return sum / float64(len(input)), nil
}

// Percentile returns the percentile
func Percentile(input []float64, percent float64) (percentile float64, err error) {
	length := len(input)
	if length == 0 {
		return math.NaN(), errors.New("The array given is empty")
	}

	if length == 1 {
		return input[0], nil
	}

	if percent <= 0 || percent > 100 {
		return math.NaN(), errors.New("Invalid given percentage")
	}

	c := sortedCopy(input)
	index := (percent / 100) * float64(len(c)) // Multiply percent by length of input

	// Check if the index is a whole number
	if index == float64(int64(index)) {
		i := int(index)
		percentile = c[i-1] // Find the value at the index
	} else if index > 1 {
		i := int(index)
		// Find the average of the index and following values
		percentile, _ = Mean([]float64{c[i-1], c[i]})
	} else {
		return math.NaN(), errors.New("Out of scope")
	}

	return percentile, nil
}
