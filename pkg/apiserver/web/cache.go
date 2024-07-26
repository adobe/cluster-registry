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
	"time"
)

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

// Response is the cached response data structure.
type Response struct {
	// Value is the cached response value.
	Value []byte

	// Header is the cached response header.
	Header http.Header

	// Expiration is the cached response expiration date.
	Expiration time.Time

	// LastAccess is the last date a cached response was accessed.
	// Used by LRU and MRU algorithms.
	LastAccess time.Time

	// Frequency is the count of times a cached response is accessed.
	// Used for LFU and MFU algorithms.
	Frequency int
}

func HTTPCache(client *cache.Cache[[]byte], appConfig *config.AppConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method == http.MethodGet {
				sortURLParams(c.Request().URL)
				key := generateKey(c.Request().URL.String())

				b, err := client.Get(c.Request().Context(), key)
				if err == nil {
					response := BytesToResponse(b)
					if response.Expiration.After(time.Now()) {
						response.LastAccess = time.Now()
						response.Frequency++
						err := client.Set(c.Request().Context(), key, response.Bytes(), store.WithExpiration(response.Expiration.Sub(time.Now())))
						if err != nil {
							c.Logger().Errorf("Error setting cache key: %s", err.Error())
							c.Error(err)
						}

						for k, v := range response.Header {
							c.Response().Header().Set(k, strings.Join(v, ","))
						}
						c.Response().WriteHeader(http.StatusOK)
						c.Response().Write(response.Value)

						return nil
					}

					err = client.Delete(c.Request().Context(), key)
					if err != nil {
						c.Logger().Errorf("Error deleting cache key: %s", err.Error())
						c.Error(err)
					}
				}

				resBody := new(bytes.Buffer)
				mw := io.MultiWriter(c.Response().Writer, resBody)
				writer := &bodyDumpResponseWriter{Writer: mw, ResponseWriter: c.Response().Writer}
				c.Response().Writer = writer
				if err := next(c); err != nil {
					c.Error(err)
				}

				statusCode := writer.statusCode
				value := resBody.Bytes()
				if statusCode < 400 {
					now := time.Now()

					response := Response{
						Value:      value,
						Header:     writer.Header(),
						Expiration: now.Add(appConfig.ApiCacheTTL),
						LastAccess: now,
						Frequency:  1,
					}
					err := client.Set(c.Request().Context(), key, response.Bytes(), store.WithExpiration(response.Expiration.Sub(time.Now())))
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

// BytesToResponse converts bytes array into Response data structure.
func BytesToResponse(b []byte) Response {
	var r Response
	dec := gob.NewDecoder(bytes.NewReader(b))
	dec.Decode(&r)

	return r
}

// Bytes converts Response data structure into bytes array.
func (r Response) Bytes() []byte {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	enc.Encode(&r)

	return b.Bytes()
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

// KeyAsString can be used by adapters to convert the cache key from uint64 to string.
func KeyAsString(key uint64) string {
	return strconv.FormatUint(key, 36)
}

func generateKey(URL string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(URL))

	return hash.Sum64()
}

func generateKeyWithBody(URL string, body []byte) uint64 {
	hash := fnv.New64a()
	body = append([]byte(URL), body...)
	hash.Write(body)

	return hash.Sum64()
}
