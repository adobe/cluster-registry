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
package monitoring

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func (m *Metrics) Use(e *echo.Echo) {
	e.Use(m.handlerFunc)
}

// handlerFunc calculate metrics for echo middleware requests
func (m *Metrics) handlerFunc(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		if c.Path() == "/metrics" {
			return next(c)
		}

		start := time.Now()
		err := next(c)
		elapsed := float64(time.Since(start)) / float64(time.Second)

		method := c.Request().Method
		status := c.Response().Status
		if err != nil {
			var httpError *echo.HTTPError
			if errors.As(err, &httpError) {
				status = httpError.Code
			}
			if status == 0 || status == http.StatusOK {
				status = http.StatusInternalServerError
			}
		}

		url := requestCounterURLLabelMappingFunc(c)
		if len(m.URLLabelFromContext) > 0 {
			u := c.Get(m.URLLabelFromContext)
			if u == nil {
				u = "unknown"
			}
			url = u.(string)
		}

		statusStr := strconv.Itoa(status)
		m.RecordIngressRequestCnt(statusStr, method, url)
		m.RecordIngressRequestDur(statusStr, method, url, elapsed)

		return err
	}
}

func requestCounterURLLabelMappingFunc(c echo.Context) string {
	return c.Path()
}
