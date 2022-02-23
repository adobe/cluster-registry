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

import (
	"fmt"
	"regexp"
)

// Convert cluster shortName into standard name format
// Ex. cluster01produseast1 becomes cluster01-prod-useast1
func GetClusterDashName(shortName string) (string, error) {
	re := regexp.MustCompile(`(^[a-z]{2,10}[0-9]{1,5})(sbx|thrash|dev|stage|prod)(.*$)`)
	result := re.FindStringSubmatch(shortName)

	if len(result) < 4 {
		return "", fmt.Errorf("Cannot convert shortName %s to standard name", shortName)
	}

	name := result[1] + "-" + result[2] + "-" + result[3]
	return name, nil
}
