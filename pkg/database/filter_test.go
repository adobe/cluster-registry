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

package database

import (
	"fmt"
	"github.com/adobe/cluster-registry/pkg/apiserver/models"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestNewDynamoDBFilter(t *testing.T) {
	test := assert.New(t)

	f := NewDynamoDBFilter()
	test.NotNil(f)
}

func TestBuild(t *testing.T) {
	test := assert.New(t)

	testCases := []struct {
		name               string
		filter             *DynamoDBFilter
		expectedError      error
		expectedExpression expression.ConditionBuilder
	}{
		{
			name: "single valid condition with equals operand",
			filter: NewDynamoDBFilter().
				AddCondition(models.NewFilterCondition("foo.bar", "=", "test")),
			expectedError: nil,
			expectedExpression: expression.Name("status").NotEqual(expression.Value("")).
				And(expression.Name("crd.spec.foo.bar").Equal(expression.Value("test"))),
		},
		{
			name: "multiple valid conditions",
			filter: NewDynamoDBFilter().
				AddCondition(models.NewFilterCondition("foo.bar", "=", "test")).
				AddCondition(models.NewFilterCondition("some.date.a", "<", "2022-05-05T00:00:00Z")).
				AddCondition(models.NewFilterCondition("some.date.b", "<=", "2022-05-05T00:00:00Z")).
				AddCondition(models.NewFilterCondition("some.date.c", ">=", "2022-05-05T00:00:00Z")).
				AddCondition(models.NewFilterCondition("crd.spec.some.date.d", ">", "2022-05-05T00:00:00Z")),
			expectedError: nil,
			expectedExpression: expression.Name("status").NotEqual(expression.Value("")).
				And(expression.Name("crd.spec.foo.bar").Equal(expression.Value("test"))).
				And(expression.Name("crd.spec.some.date.a").LessThan(expression.Value("2022-05-05 00:00:00 +0000 UTC"))).
				And(expression.Name("crd.spec.some.date.b").LessThanEqual(expression.Value("2022-05-05 00:00:00 +0000 UTC"))).
				And(expression.Name("crd.spec.some.date.c").GreaterThanEqual(expression.Value("2022-05-05 00:00:00 +0000 UTC"))).
				And(expression.Name("crd.spec.some.date.d").GreaterThan(expression.Value("2022-05-05 00:00:00 +0000 UTC"))),
		},
		{
			name: "single valid condition with full field name",
			filter: NewDynamoDBFilter().
				AddCondition(models.NewFilterCondition("crd.spec.foo.bar", "=", "test")),
			expectedError: nil,
			expectedExpression: expression.Name("status").NotEqual(expression.Value("")).
				And(expression.Name("crd.spec.foo.bar").Equal(expression.Value("test"))),
		},
		{
			name: "condition with invalid operand",
			filter: NewDynamoDBFilter().
				AddCondition(models.NewFilterCondition("foo.bar", "_", "test")),
			expectedError: fmt.Errorf("failed to parse operand _: invalid operand, must use one of %s", strings.Join(models.AllowedOperands, ", ")),
		},
		{
			name: "single valid condition comparison",
			filter: NewDynamoDBFilter().
				AddCondition(models.NewFilterCondition("some.date", ">", "2022-05-05T00:00:00Z")),
			expectedError: nil,
			expectedExpression: expression.Name("status").NotEqual(expression.Value("")).
				And(expression.Name("crd.spec.some.date").GreaterThan(expression.Value("2022-05-05 00:00:00 +0000 UTC"))),
		},
	}

	for _, tc := range testCases {
		expr, err := tc.filter.Build()
		test.Equal(tc.expectedError, err)
		if tc.expectedError == nil && test.NoError(err) {
			test.Equal(tc.expectedExpression, expr)
		}
	}
}
