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

package web

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClusterDashName(t *testing.T) {
	test := assert.New(t)
	tcs := []struct {
		name          string
		shortName     string
		expectedName  string
		expectedError error
	}{
		{
			name:          "valid standard shortname",
			shortName:     "cluster01produseast1",
			expectedName:  "cluster01-prod-useast1",
			expectedError: nil,
		},
		{
			name:          "invalid shortname",
			shortName:     "clusterproduseast1",
			expectedName:  "",
			expectedError: fmt.Errorf("Cannot convert shortName"),
		},
		{
			name:          "invalid shortname environment",
			shortName:     "cluster01nonameuseast1",
			expectedName:  "",
			expectedError: fmt.Errorf("Cannot convert shortName"),
		},
		{
			name:          "valid big number shortname",
			shortName:     "cluster9999produseast1",
			expectedName:  "cluster9999-prod-useast1",
			expectedError: nil,
		},
	}
	for _, tc := range tcs {
		t.Logf("\tTest %s:\tWhen converting cluster %s to %s and expecting error %v", tc.name, tc.shortName, tc.expectedName, tc.expectedError)

		name, err := GetClusterDashName(tc.shortName)

		if tc.expectedError != nil {
			test.Contains(err.Error(), tc.expectedError.Error())
			continue
		}
		test.NoError(err)
		assert.Equal(t, tc.expectedName, name)
	}
}
