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

package models

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFilterConditionFromQuery(t *testing.T) {
	test := assert.New(t)

	successTestCases := []struct {
		query                                         string
		expectedField, expectedOperand, expectedValue string
	}{
		{
			"foo.bar:<10",
			"foo.bar", "<", "10",
		},
		{
			"foo.bar:=",
			"foo.bar", "=", "",
		},
	}

	failureTestCases := []struct {
		query         string
		expectedError error
	}{
		{
			"foo.bar :<10",
			fmt.Errorf("invalid query"),
		},
		{
			"foo.bar: < 10",
			fmt.Errorf("invalid query"),
		},
		{
			"foo.bar:*10",
			fmt.Errorf("invalid query"),
		},
	}

	for _, tc := range successTestCases {
		condition, err := NewFilterConditionFromQuery(tc.query)
		test.NoError(err)
		if test.NotNil(condition) {
			test.Equal(tc.expectedField, condition.Field)
			test.Equal(tc.expectedOperand, condition.Operand)
			test.Equal(tc.expectedValue, condition.Value)
		}
	}

	for _, tc := range failureTestCases {
		condition, err := NewFilterConditionFromQuery(tc.query)
		test.Error(tc.expectedError, err)
		test.Nil(condition)
	}
}
