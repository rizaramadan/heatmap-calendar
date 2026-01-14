package middleware

import (
	"net/http"

	"github.com/gti/heatmap-internal/internal/service"
	"github.com/labstack/echo/v4"
)

const (
	SessionCookieName = "session_token"
	UserEmailKey      = "user_email"
)

// SessionAuth returns middleware that validates session cookies
func SessionAuth(authService *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if already authenticated from SessionAuthOptional
			if email := GetUserEmail(c); email != "" {
				return next(c)
			}

			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "not authenticated",
				})
			}

			email, err := authService.ValidateSession(c.Request().Context(), cookie.Value)
			if err != nil {
				// Clear invalid cookie
				c.SetCookie(&http.Cookie{
					Name:     SessionCookieName,
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "session expired or invalid",
				})
			}

			// Store user email in context
			c.Set(UserEmailKey, email)

			return next(c)
		}
	}
}

// SessionAuthOptional is like SessionAuth but doesn't reject unauthenticated requests
func SessionAuthOptional(authService *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				// No cookie, continue without auth
				return next(c)
			}

			email, err := authService.ValidateSession(c.Request().Context(), cookie.Value)
			if err != nil {
				// Invalid session, continue without auth
				return next(c)
			}

			// Store user email in context
			c.Set(UserEmailKey, email)

			return next(c)
		}
	}
}

// GetUserEmail returns the authenticated user's email from context
func GetUserEmail(c echo.Context) string {
	email, _ := c.Get(UserEmailKey).(string)
	return email
}

// IsAuthenticated returns whether the request has a valid session
func IsAuthenticated(c echo.Context) bool {
	return GetUserEmail(c) != ""
}

// SetSessionCookie sets the session cookie
func SetSessionCookie(c echo.Context, token string) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
