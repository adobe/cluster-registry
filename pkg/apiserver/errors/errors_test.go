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

package errors

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	test := assert.New(t)

	t.Log("Test creating new error.")

	tcs := []struct {
		name          string
		inputError    Error
		expectedError Error
	}{
		{
			name:          "simple error",
			inputError:    NewError(fmt.Errorf("Validation failed")),
			expectedError: Error{Errors: map[string]interface{}{"body": "Validation failed"}},
		},
		{
			name:          "http error",
			inputError:    NewError(echo.NewHTTPError(http.StatusBadGateway, "Bad gateway")),
			expectedError: Error{Errors: map[string]interface{}{"body": "Bad gateway"}},
		},
		{
			name:          "simple error",
			inputError:    NotFound(),
			expectedError: Error{Errors: map[string]interface{}{"body": "resource not found"}},
		},
	}

	for _, tc := range tcs {
		t.Logf("\tTest %s:\tWhen creating new error with input: %v", tc.name, tc.inputError)

		test.Contains(fmt.Sprintf("%v", tc.inputError), fmt.Sprintf("%v", tc.expectedError), "the error message should be as expected")
	}
}
