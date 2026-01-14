package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
)

type CapacityService struct {
	entityRepo   *repository.EntityRepository
	capacityRepo *repository.CapacityRepository
}

func NewCapacityService(
	entityRepo *repository.EntityRepository,
	capacityRepo *repository.CapacityRepository,
) *CapacityService {
	return &CapacityService{
		entityRepo:   entityRepo,
		capacityRepo: capacityRepo,
	}
}

// UpdateDefaultCapacity updates the default capacity for an entity
func (s *CapacityService) UpdateDefaultCapacity(ctx context.Context, entityID string, capacity float64) error {
	if capacity < 0 {
		return fmt.Errorf("capacity cannot be negative")
	}

	return s.entityRepo.UpdateDefaultCapacity(ctx, entityID, capacity)
}

// SetDateOverride sets a capacity override for a specific date
func (s *CapacityService) SetDateOverride(ctx context.Context, entityID string, date time.Time, capacity float64) error {
	if capacity < 0 {
		return fmt.Errorf("capacity cannot be negative")
	}

	// Verify entity exists
	_, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("failed to verify entity: %w", err)
	}

	override := &models.CapacityOverride{
		EntityID: entityID,
		Date:     date,
		Capacity: capacity,
	}

	return s.capacityRepo.SetOverride(ctx, override)
}

// DeleteDateOverride removes a capacity override for a specific date
func (s *CapacityService) DeleteDateOverride(ctx context.Context, entityID string, date time.Time) error {
	return s.capacityRepo.DeleteOverride(ctx, entityID, date)
}

// GetCapacityInfo returns capacity information for an entity
func (s *CapacityService) GetCapacityInfo(ctx context.Context, entityID string) (*models.Entity, []models.CapacityOverride, error) {
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get entity: %w", err)
	}

	// Get overrides for the next 90 days
	today := time.Now().Truncate(24 * time.Hour)
	endDate := today.AddDate(0, 0, 90)

	overrides, err := s.capacityRepo.GetOverridesRange(ctx, entityID, today, endDate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get overrides: %w", err)
	}

	return entity, overrides, nil
}

// UpdateCapacity handles the full capacity update request
func (s *CapacityService) UpdateCapacity(ctx context.Context, entityID string, req *models.UpdateCapacityRequest) error {
	// Update default capacity if provided
	if req.DefaultCapacity != nil {
		if err := s.UpdateDefaultCapacity(ctx, entityID, *req.DefaultCapacity); err != nil {
			return fmt.Errorf("failed to update default capacity: %w", err)
		}
	}

	// Process date overrides
	for _, override := range req.DateOverrides {
		date, err := time.Parse("2006-01-02", override.Date)
		if err != nil {
			return fmt.Errorf("invalid date format for %s: %w", override.Date, err)
		}

		if err := s.SetDateOverride(ctx, entityID, date, override.Capacity); err != nil {
			return fmt.Errorf("failed to set override for %s: %w", override.Date, err)
		}
	}

	return nil
}
