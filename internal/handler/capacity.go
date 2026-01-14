package handler

import (
	"html/template"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gti/heatmap-internal/internal/middleware"
	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/service"
	"github.com/labstack/echo/v4"
)

type CapacityHandler struct {
	capacityService *service.CapacityService
	templates       *template.Template
	validate        *validator.Validate
}

func NewCapacityHandler(
	capacityService *service.CapacityService,
	templates *template.Template,
) *CapacityHandler {
	return &CapacityHandler{
		capacityService: capacityService,
		templates:       templates,
		validate:        validator.New(),
	}
}

// MyCapacityPage renders the capacity management form for the logged-in user
func (h *CapacityHandler) MyCapacityPage(c echo.Context) error {
	userEmail := middleware.GetUserEmail(c)
	log.Printf("MyCapacityPage: userEmail from context = '%s'", userEmail)
	if userEmail == "" {
		log.Printf("MyCapacityPage: No userEmail, redirecting to /login")
		return c.Redirect(http.StatusFound, "/login")
	}

	entity, overrides, err := h.capacityService.GetCapacityInfo(c.Request().Context(), userEmail)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load capacity data")
	}

	data := map[string]interface{}{
		"Entity":          entity,
		"Overrides":       overrides,
		"IsAuthenticated": true,
		"UserEmail":       userEmail,
	}

	return h.templates.ExecuteTemplate(c.Response().Writer, "capacity_form", data)
}

// UpdateMyCapacity handles the capacity update request for the logged-in user
func (h *CapacityHandler) UpdateMyCapacity(c echo.Context) error {
	userEmail := middleware.GetUserEmail(c)
	if userEmail == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
	}

	var req models.UpdateCapacityRequest
	if err := c.Bind(&req); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusBadRequest, `<div class="text-red-500">Invalid request</div>`)
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if err := h.capacityService.UpdateCapacity(c.Request().Context(), userEmail, &req); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return c.HTML(http.StatusInternalServerError, `<div class="text-red-500">Failed to update capacity</div>`)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return c.HTML(http.StatusOK, `<div class="text-green-500">Capacity updated successfully!</div>`)
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "capacity updated"})
}

// GetCapacityForm returns the capacity form partial (HTMX)
func (h *CapacityHandler) GetCapacityForm(c echo.Context) error {
	userEmail := middleware.GetUserEmail(c)
	if userEmail == "" {
		return c.HTML(http.StatusUnauthorized, `<div class="text-red-500">Please log in</div>`)
	}

	entity, overrides, err := h.capacityService.GetCapacityInfo(c.Request().Context(), userEmail)
	if err != nil {
		return c.HTML(http.StatusInternalServerError, `<div class="text-red-500">Failed to load data</div>`)
	}

	data := map[string]interface{}{
		"Entity":    entity,
		"Overrides": overrides,
	}

	return h.templates.ExecuteTemplate(c.Response().Writer, "capacity_form_partial.html", data)
}
