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

package database

import (
	"fmt"
	"github.com/adobe/cluster-registry/pkg/apiserver/models"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"strings"
	"time"
)

const FieldPrefix = "crd.spec."

type DynamoDBFilter struct {
	conditions []models.FilterCondition
}

func NewDynamoDBFilter() *DynamoDBFilter {
	return &DynamoDBFilter{
		conditions: []models.FilterCondition{},
	}
}

func (f *DynamoDBFilter) Build() (expression.ConditionBuilder, error) {
	var filter expression.ConditionBuilder

	filter = expression.Name("status").NotEqual(expression.Value(""))

	for _, c := range f.conditions {
		field, err := parseField(c.Field)
		if err != nil {
			return filter, fmt.Errorf("failed to parse field %s: %v", c.Field, err)
		}

		operand, err := parseOperand(c.Operand)
		if err != nil {
			return filter, fmt.Errorf("failed to parse operand %s: %v", c.Operand, err)
		}

		value, err := parseValue(c.Value, c.Operand)
		if err != nil {
			return filter, fmt.Errorf("failed to parse value %s: %v", c.Value, err)
		}

		switch operand {
		case "=":
			filter = filter.And(field.Equal(value))
		case ">=":
			filter = filter.And(field.GreaterThanEqual(value))
		case ">":
			filter = filter.And(field.GreaterThan(value))
		case "<=":
			filter = filter.And(field.LessThanEqual(value))
		case "<":
			filter = filter.And(field.LessThan(value))
		}
	}

	return filter, nil
}

func (f *DynamoDBFilter) AddCondition(condition *models.FilterCondition) *DynamoDBFilter {
	f.conditions = append(f.conditions, *condition)
	return f
}

func contains(item string, slice []string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func parseField(field string) (expression.NameBuilder, error) {
	// TODO: check if field is a valid nested property of Cluster CRD
	if !strings.HasPrefix(field, FieldPrefix) {
		field = fmt.Sprintf("%s%s", FieldPrefix, field)
	}
	return expression.Name(field), nil
}

func parseOperand(operand string) (string, error) {
	if !contains(operand, models.AllowedOperands) {
		return operand, fmt.Errorf("invalid operand, must use one of %s", strings.Join(models.AllowedOperands, ", "))
	}
	return operand, nil
}

func parseValue(value, operand string) (expression.ValueBuilder, error) {
	if contains(operand, []string{"<", "<=", ">=", ">"}) {
		date, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return expression.Value(date.String()), nil
		}
	}

	return expression.Value(value), nil
}
