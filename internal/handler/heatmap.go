package handler

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gti/heatmap-internal/internal/middleware"
	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
	"github.com/gti/heatmap-internal/internal/service"
	"github.com/labstack/echo/v4"
)

type HeatmapHandler struct {
	heatmapService *service.HeatmapService
	entityRepo     *repository.EntityRepository
	templates      *template.Template
}

func NewHeatmapHandler(
	heatmapService *service.HeatmapService,
	entityRepo *repository.EntityRepository,
	templates *template.Template,
) *HeatmapHandler {
	return &HeatmapHandler{
		heatmapService: heatmapService,
		entityRepo:     entityRepo,
		templates:      templates,
	}
}

// Index renders the main heatmap page
func (h *HeatmapHandler) Index(c echo.Context) error {
	entityID := c.QueryParam("entity")

	// Get list of all entities for the selector
	entities, err := h.entityRepo.ListAll(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load entities")
	}

	data := map[string]interface{}{
		"Entities":        entities,
		"SelectedEntity":  entityID,
		"IsAuthenticated": middleware.IsAuthenticated(c),
		"UserEmail":       middleware.GetUserEmail(c),
	}

	// If entity is selected, load heatmap data
	if entityID != "" {
		heatmapData, err := h.heatmapService.GetHeatmapData(c.Request().Context(), entityID, 90)
		if err != nil {
			data["Error"] = "Failed to load heatmap data"
		} else {
			data["HeatmapData"] = heatmapData
			data["Months"] = groupDaysByMonth(heatmapData.Days)
		}
	}

	return h.templates.ExecuteTemplate(c.Response().Writer, "heatmap", data)
}

// GetHeatmapPartial returns the heatmap grid as an HTMX partial
func (h *HeatmapHandler) GetHeatmapPartial(c echo.Context) error {
	entityID := c.Param("entity")

	heatmapData, err := h.heatmapService.GetHeatmapData(c.Request().Context(), entityID, 90)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load heatmap")
	}

	data := map[string]interface{}{
		"HeatmapData": heatmapData,
		"Months":      groupDaysByMonth(heatmapData.Days),
		"EntityID":    entityID,
	}

	return h.templates.ExecuteTemplate(c.Response().Writer, "heatmap_grid.html", data)
}

// GetDayDetails returns the tasks/loads for a specific day (HTMX partial)
func (h *HeatmapHandler) GetDayDetails(c echo.Context) error {
	entityID := c.Param("entity")
	dateStr := c.Param("date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid date format")
	}

	loads, totalLoad, capacity, err := h.heatmapService.GetDayDetails(c.Request().Context(), entityID, date)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load day details")
	}

	data := map[string]interface{}{
		"Date":      date,
		"DateStr":   dateStr,
		"Loads":     loads,
		"TotalLoad": totalLoad,
		"Capacity":  capacity,
		"EntityID":  entityID,
	}

	return h.templates.ExecuteTemplate(c.Response().Writer, "day_tasks", data)
}

// MonthData represents grouped days for a month
type MonthData struct {
	Year      int
	Month     time.Month
	MonthName string
	Days      []DayData
}

// DayData represents a single day in the heatmap
type DayData struct {
	Date     time.Time
	DateStr  string
	Day      int
	Load     float64
	Capacity float64
	Color    string
	IsToday  bool
}

// groupDaysByMonth groups heatmap days by month for template rendering
func groupDaysByMonth(days []models.HeatmapDay) []MonthData {
	today := time.Now().Truncate(24 * time.Hour)
	monthMap := make(map[string]*MonthData)
	var monthOrder []string

	for _, day := range days {
		key := day.Date.Format("2006-01")
		if _, exists := monthMap[key]; !exists {
			monthMap[key] = &MonthData{
				Year:      day.Date.Year(),
				Month:     day.Date.Month(),
				MonthName: day.Date.Month().String(),
				Days:      []DayData{},
			}
			monthOrder = append(monthOrder, key)
		}

		monthMap[key].Days = append(monthMap[key].Days, DayData{
			Date:     day.Date,
			DateStr:  day.Date.Format("2006-01-02"),
			Day:      day.Date.Day(),
			Load:     day.Load,
			Capacity: day.Capacity,
			Color:    day.Color,
			IsToday:  day.Date.Equal(today),
		})
	}

	result := make([]MonthData, 0, len(monthOrder))
	for _, key := range monthOrder {
		result = append(result, *monthMap[key])
	}

	return result
}
