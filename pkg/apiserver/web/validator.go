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

package web

import "gopkg.in/go-playground/validator.v9"

// NewValidator func
func NewValidator() *Validator {
	return &Validator{
		validator: validator.New(),
	}
}

// Validator func
type Validator struct {
	validator *validator.Validate
}

// Validate func
func (v *Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}
