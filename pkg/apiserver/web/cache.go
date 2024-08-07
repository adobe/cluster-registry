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
	"encoding/gob"
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
	"strings"
)

type Response struct {
	// Value is the cached response value.
	Value []byte

	// Header is the cached response header.
	Header http.Header
}

func StringToResponse(s string) Response {
	var r Response
	dec := gob.NewDecoder(strings.NewReader(s))
	_ = dec.Decode(&r)

	return r
}

func (r Response) String() string {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	_ = enc.Encode(r)

	return b.String()
}

func HTTPCache(client *cache.Cache[string], appConfig *config.AppConfig, tags []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method == http.MethodGet {
				sortURLParams(c.Request().URL)
				key := GenerateKey(c.Request().URL.String())

				cachedResponse, err := client.Get(c.Request().Context(), key)
				response := StringToResponse(cachedResponse)
				if err != nil {
					c.Logger().Warnf("Error getting key from cache: %s", err.Error())
				}

				// if key in cache
				if cachedResponse != "" {
					// return body from cache
					for k, v := range response.Header {
						c.Response().Header().Set(k, strings.Join(v, ","))
					}
					c.Response().WriteHeader(http.StatusOK)
					_, _ = c.Response().Write(response.Value)

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
					newResponse := Response{
						Value:  resBody.Bytes(),
						Header: writer.Header(),
					}
					err := client.Set(c.Request().Context(), key, newResponse.String(), store.WithExpiration(appConfig.ApiCacheTTL), store.WithTags(tags))
					if err != nil {
						c.Logger().Errorf("Error setting cache key: %s", err.Error())
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
	_, err := hash.Write([]byte(URL))
	if err != nil {
		return ""
	}

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
