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
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/adobe/cluster-registry/test/slt/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewEchoWithLogger returns an echo server with a specific logger
func NewEchoWithLogger(logger *log.Logger) *echo.Echo {
	e := echo.New()
	e.Logger = logger // App logs

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger()) // Logs http requests

	return e
}

// Metrics returns a HandlerFunc for the metrics endpoint
func Metrics() echo.HandlerFunc {
	h := promhttp.Handler()
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// Livez checks if the apiserver is up
func Livez(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

// WaitForShutdownSignal waits for interrupt signal to gracefully shutdown the server.
func WaitForShutdownSignal(e *echo.Echo) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-quit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.Logger.Info("Gracefully shuting down the server.")
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
