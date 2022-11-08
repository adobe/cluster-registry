package models

import (
	"fmt"
	"regexp"
	"strings"
)

// AllowedOperands ...
var AllowedOperands = []string{"<", "<=", "=", ">=", ">"}

type Filter interface {
	Build() (interface{}, error)
	AddCondition(condition FilterCondition) error
	Conditions() []FilterCondition
}

type FilterCondition struct {
	Field   string
	Operand string
	Value   string
}

func NewFilterCondition(field, operand, value string) *FilterCondition {
	return &FilterCondition{field, operand, value}
}

func NewFilterConditionFromQuery(query string) (*FilterCondition, error) {
	// Query syntax:
	// <field>:<operand><value>

	re, _ := regexp.Compile(fmt.Sprintf(`([a-zA-Z0-9-_.]+):(%s)(.*)`, strings.Join(AllowedOperands, "|")))
	match := re.FindStringSubmatch(query)

	if len(match) == 0 {
		return nil, fmt.Errorf("invalid query")
	}

	field, operand, value := match[1], match[2], strings.TrimSpace(match[3])
	return NewFilterCondition(field, operand, value), nil
}
