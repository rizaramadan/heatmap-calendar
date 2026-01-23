package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
)

type HeatmapService struct {
	entityRepo   *repository.EntityRepository
	capacityRepo *repository.CapacityRepository
	loadRepo     *repository.LoadRepository
	groupRepo    *repository.GroupRepository
}

func NewHeatmapService(
	entityRepo *repository.EntityRepository,
	capacityRepo *repository.CapacityRepository,
	loadRepo *repository.LoadRepository,
	groupRepo *repository.GroupRepository,
) *HeatmapService {
	return &HeatmapService{
		entityRepo:   entityRepo,
		capacityRepo: capacityRepo,
		loadRepo:     loadRepo,
		groupRepo:    groupRepo,
	}
}

// GetHeatmapData returns heatmap data for an entity spanning 1 month previous and 6 months ahead from today
func (s *HeatmapService) GetHeatmapData(ctx context.Context, entityID string, days int) (*models.HeatmapData, error) {
	// Get the entity
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Calculate date range: 1 month previous and 6 months ahead
	// Use UTC for consistent date handling
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startDate := today.AddDate(0, -1, 0) // 1 month before today
	endDate := today.AddDate(0, 6, 0)    // 6 months after today

	// Get capacities for the date range
	capacities, err := s.capacityRepo.GetCapacitiesForRange(ctx, entityID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get capacities: %w", err)
	}

	// Get loads based on entity type
	var loads map[time.Time]float64
	if entity.Type == models.EntityTypePerson {
		loads, err = s.loadRepo.GetPersonLoadForDateRange(ctx, entityID, startDate, endDate)
	} else {
		loads, err = s.loadRepo.GetGroupLoadForDateRange(ctx, entityID, startDate, endDate)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get loads: %w", err)
	}

	// Build heatmap days
	heatmapDays := make([]models.HeatmapDay, 0, 300)
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// Use UTC date for lookup
		lookupDate := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		load := loads[lookupDate]
		capacity := capacities[lookupDate]
		color := getHeatmapColor(load, capacity)

		heatmapDays = append(heatmapDays, models.HeatmapDay{
			Date:     d,
			Load:     load,
			Capacity: capacity,
			Color:    color,
		})
	}

	return &models.HeatmapData{
		Entity: *entity,
		Days:   heatmapDays,
	}, nil
}

// GetDayDetails returns detailed load information for a specific day
func (s *HeatmapService) GetDayDetails(ctx context.Context, entityID string, date time.Time) ([]models.LoadWithAssignments, float64, float64, error) {
	// Get entity
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get loads for this date
	loads, err := s.loadRepo.GetLoadsForEntityOnDate(ctx, entityID, entity.Type, date)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get loads: %w", err)
	}

	// Calculate total load
	var totalLoad float64
	for _, l := range loads {
		for _, a := range l.Assignments {
			totalLoad += a.Weight
		}
	}

	// Get capacity
	capacity, err := s.capacityRepo.GetEffectiveCapacity(ctx, entityID, date)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get capacity: %w", err)
	}

	return loads, totalLoad, capacity, nil
}

// getHeatmapColor returns the appropriate color based on load/capacity ratio
func getHeatmapColor(load, capacity float64) string {
	if capacity == 0 {
		if load > 0 {
			return "#8B0000" // Blood red - any load with zero capacity is overloaded
		}
		return "#e5e7eb" // Gray for zero capacity, zero load
	}

	ratio := load / capacity
	switch {
	case ratio > 1.0:
		return "#8B0000" // Blood red - overloaded
	case ratio > 0.8:
		return "#dc2626" // Red - near capacity
	case ratio > 0.6:
		return "#f97316" // Orange
	case ratio > 0.4:
		return "#fbbf24" // Yellow/Amber
	case ratio > 0.2:
		return "#a3e635" // Lime green
	case ratio > 0:
		return "#22c55e" // Green - low load
	default:
		return "#e5e7eb" // Gray - no load
	}
}

// GetHeatmapColorForValues is exported for use in templates
func GetHeatmapColorForValues(load, capacity float64) string {
	return getHeatmapColor(load, capacity)
}
