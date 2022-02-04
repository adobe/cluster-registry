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

package api

import (
	"net/http"

	"github.com/adobe/cluster-registry/pkg/api/database"
	"github.com/adobe/cluster-registry/pkg/api/monitoring"
	"github.com/adobe/cluster-registry/pkg/api/sqs"
	"github.com/adobe/cluster-registry/pkg/api/utils"

	"github.com/labstack/echo/v4"
)

type status struct {
	Database bool `json:"database"`
	Sqs      bool `json:"sqs"`
}

// StatusSessions is used to keep the same objects and state for the database
// and sqs that are used for the rest of the calls inside the project
type StatusSessions struct {
	Consumer  sqs.Consumer
	Db        database.Db
	AppConfig *utils.AppConfig
	Metrics   monitoring.MetricsI
}

func (s *StatusSessions) checkDBStatus() bool {
	if err := s.Db.Status(); err != nil {
		return false
	}
	return true
}

func (s *StatusSessions) checkSqsStatus() bool {
	if err := s.Consumer.Status(s.AppConfig, s.Metrics); err != nil {
		return false
	}
	return true
}

// ServiceStatus checks if the services that the api uses are healthy
func (s *StatusSessions) ServiceStatus(c echo.Context) error {

	statusResponse := status{
		Database: s.checkDBStatus(),
		Sqs:      s.checkSqsStatus(),
	}

	return c.JSON(http.StatusOK, statusResponse)
}
