package web

import (
	"net/http"
	"time"

	"github.com/adobe/cluster-registry/pkg/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RateLimiter returns a middleware.RateLimiterWithConfig
func RateLimiter(appConfig *config.AppConfig) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			return appConfig.ApiRateLimiterEnabled
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: 2, Burst: 120, ExpiresIn: 1 * time.Minute},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			oid, ok := ctx.Get("oid").(string)
			if !ok {
				return "00000000-0000-0000-0000-000000000000", nil
			}
			return oid, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	})
}
