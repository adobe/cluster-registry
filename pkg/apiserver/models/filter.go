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
	"regexp"
	"strings"
)

// AllowedOperands ...
var AllowedOperands = []string{"<", "<=", "=", ">=", ">"}

type Filter interface {
	Build() (interface{}, error)
	AddCondition(condition FilterCondition) error
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
