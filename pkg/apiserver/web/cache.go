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
	"bufio"
	"bytes"
	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/labstack/echo/v4"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
)

func HTTPCache(client *cache.Cache[string], appConfig *config.AppConfig, tags []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method == http.MethodGet {
				sortURLParams(c.Request().URL)
				key := GenerateKey(c.Request().URL.String())

				value, err := client.Get(c.Request().Context(), key)
				if err != nil {
					c.Logger().Errorf("Error getting key from cache: %s", err.Error())
					c.Error(err)
				}

				// if key in cache
				if value != "" {
					// return body from cache
					_, _ = c.Response().Write([]byte(value))
					return nil
				}

				// if key not in cache then write response to cache
				resBody := new(bytes.Buffer)
				mw := io.MultiWriter(c.Response().Writer, resBody)
				writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
				c.Response().Writer = writer
				if err := next(c); err != nil {
					c.Error(err)
				}
				if writer.statusCode < 400 {
					err := client.Set(c.Request().Context(), key, resBody.String(), store.WithExpiration(appConfig.ApiCacheTTL), store.WithTags(tags))
					if err != nil {
						c.Logger().Errorf("Error setting cache key: %s", err.Error())
						c.Error(err)
					}
				}
				return nil
			}

			if err := next(c); err != nil {
				c.Error(err)
			}
			return nil
		}
	}
}

func sortURLParams(URL *url.URL) {
	params := URL.Query()
	for _, param := range params {
		sort.Slice(param, func(i, j int) bool {
			return param[i] < param[j]
		})
	}
	URL.RawQuery = params.Encode()
}

func GenerateKey(URL string) string {
	hash := fnv.New64a()
	hash.Write([]byte(URL))

	return strconv.FormatUint(hash.Sum64(), 36)
}

type bodyDumpResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *bodyDumpResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyDumpResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *bodyDumpResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *bodyDumpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}
