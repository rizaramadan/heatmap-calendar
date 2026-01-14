package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// APIKeyAuth returns middleware that validates the x-api-key header
func APIKeyAuth(apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip if no API key configured (development mode)
			if apiKey == "" {
				return next(c)
			}

			key := c.Request().Header.Get("x-api-key")
			if key == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "missing x-api-key header",
				})
			}

			if key != apiKey {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid API key",
				})
			}

			return next(c)
		}
	}
}
