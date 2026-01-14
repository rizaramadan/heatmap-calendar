package handler

import (
	"html/template"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gti/heatmap-internal/internal/middleware"
	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
	"github.com/gti/heatmap-internal/internal/service"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService *service.AuthService
	entityRepo  *repository.EntityRepository
	templates   *template.Template
	validate    *validator.Validate
}

func NewAuthHandler(
	authService *service.AuthService,
	entityRepo *repository.EntityRepository,
	templates *template.Template,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		entityRepo:  entityRepo,
		templates:   templates,
		validate:    validator.New(),
	}
}

// LoginPage renders the login form
func (h *AuthHandler) LoginPage(c echo.Context) error {
	// If already authenticated, redirect to home
	if middleware.IsAuthenticated(c) {
		return c.Redirect(http.StatusFound, "/")
	}

	data := map[string]interface{}{
		"Step": "email",
	}

	return h.templates.ExecuteTemplate(c.Response().Writer, "login", data)
}

// RequestOTP sends an OTP to the user's email
func (h *AuthHandler) RequestOTP(c echo.Context) error {
	var req models.OTPRequest

	// Handle both form and JSON
	if c.Request().Header.Get("Content-Type") == "application/json" {
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
	} else {
		req.Email = c.FormValue("email")
	}

	if err := h.validate.Struct(req); err != nil {
		// For HTMX, return HTML partial
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500">Please enter a valid email address</div>`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Verify user exists (must be a registered person)
	_, err := h.entityRepo.GetByID(c.Request().Context(), req.Email)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500">Email not found in system</div>`)
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": "email not found"})
	}

	// Send OTP
	if err := h.authService.SendOTP(c.Request().Context(), req.Email); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500">Failed to send code. Please try again.</div>`)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send OTP"})
	}

	// For HTMX, return the OTP verification form
	if c.Request().Header.Get("HX-Request") == "true" {
		data := map[string]interface{}{
			"Email": req.Email,
		}
		return h.templates.ExecuteTemplate(c.Response().Writer, "otp_form", data)
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "OTP sent"})
}

// VerifyOTP verifies the OTP and creates a session
func (h *AuthHandler) VerifyOTP(c echo.Context) error {
	var req models.VerifyOTPRequest

	// Handle both form and JSON
	if c.Request().Header.Get("Content-Type") == "application/json" {
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
	} else {
		req.Email = c.FormValue("email")
		req.OTP = c.FormValue("otp")
	}

	if err := h.validate.Struct(req); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500">Please enter a valid 6-digit code</div>`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Verify OTP
	valid, err := h.authService.VerifyOTP(c.Request().Context(), req.Email, req.OTP)
	if err != nil || !valid {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500">Invalid or expired code</div>`)
		}
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired OTP"})
	}

	// Create session
	token, err := h.authService.CreateSession(c.Request().Context(), req.Email)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500">Failed to create session</div>`)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	// Set session cookie
	middleware.SetSessionCookie(c, token)

	// For HTMX, redirect via header
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/")
		return c.String(http.StatusOK, "")
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "logged in"})
}

// Logout clears the session
func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie(middleware.SessionCookieName)
	if err == nil {
		h.authService.DeleteSession(c.Request().Context(), cookie.Value)
	}

	middleware.ClearSessionCookie(c)

	// For HTMX, redirect
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/")
		return c.String(http.StatusOK, "")
	}

	return c.Redirect(http.StatusFound, "/")
}
