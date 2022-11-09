package database

import (
	"errors"
	"fmt"
	"github.com/adobe/cluster-registry/pkg/apiserver/models"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"strings"
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

func (f *DynamoDBFilter) Build() (interface{}, error) {
	var filter expression.ConditionBuilder

	filter = expression.Name("status").NotEqual(expression.Value(""))

	for _, c := range f.conditions {
		field := expression.Name(fmt.Sprintf("%s%s", FieldPrefix, strings.TrimRight(c.Field, FieldPrefix)))
		value := expression.Value(c.Value)
		switch c.Operand {
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

func (f *DynamoDBFilter) AddCondition(condition *models.FilterCondition) error {
	if !contains(condition.Operand, models.AllowedOperands) {
		return errors.New(fmt.Sprintf("invalid operand, must use one of %s", strings.Join(models.AllowedOperands, ",")))
	}

	f.conditions = append(f.conditions, *condition)

	return nil
}

func (f *DynamoDBFilter) Conditions() []models.FilterCondition {
	return f.conditions
}

func contains(item string, slice []string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
