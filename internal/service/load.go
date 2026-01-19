package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
)

type LoadService struct {
	loadRepo       *repository.LoadRepository
	entityRepo     *repository.EntityRepository
	webhookService *WebhookService
}

func NewLoadService(
	loadRepo *repository.LoadRepository,
	entityRepo *repository.EntityRepository,
	webhookService *WebhookService,
) *LoadService {
	return &LoadService{
		loadRepo:       loadRepo,
		entityRepo:     entityRepo,
		webhookService: webhookService,
	}
}

// UpsertLoad creates or updates a load with its assignments
func (s *LoadService) UpsertLoad(ctx context.Context, req *models.UpsertLoadRequest) (int, error) {
	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return 0, fmt.Errorf("invalid date format: %w", err)
	}

	// Validate all assignees exist
	for _, a := range req.Assignees {
		exists, err := s.entityRepo.Exists(ctx, a.Email)
		if err != nil {
			return 0, fmt.Errorf("failed to check assignee: %w", err)
		}
		if !exists {
			return 0, fmt.Errorf("assignee not found: %s", a.Email)
		}
	}

	// Build load and assignments
	externalID := req.ExternalID
	source := req.Source
	url := req.URL
	load := &models.Load{
		ExternalID: &externalID,
		Title:      req.Title,
		Source:     &source,
		URL:        &url,
		Date:       date,
	}

	assignments := make([]models.LoadAssignment, 0, len(req.Assignees))
	for _, a := range req.Assignees {
		weight := a.Weight
		if weight == 0 {
			weight = 1.0 // Default weight
		}
		assignments = append(assignments, models.LoadAssignment{
			PersonEmail: a.Email,
			Weight:      weight,
		})
	}

	// Upsert the load
	loadID, err := s.loadRepo.UpsertByExternalID(ctx, load, assignments)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert load: %w", err)
	}

	// Trigger webhook alerts for affected persons (in background)
	for _, a := range req.Assignees {
		s.webhookService.CheckAndAlert(a.Email, date)
	}

	return loadID, nil
}

// GetLoadsByDateRange returns loads within a date range
func (s *LoadService) GetLoadsByDateRange(ctx context.Context, start, end time.Time) ([]models.LoadWithAssignments, error) {
	return s.loadRepo.GetLoadsByDateRange(ctx, start, end)
}

// DeleteLoad deletes a load by ID
func (s *LoadService) DeleteLoad(ctx context.Context, id int) error {
	return s.loadRepo.Delete(ctx, id)
}

// AddAssignees adds one or more assignees to an existing load
func (s *LoadService) AddAssignees(ctx context.Context, loadID int, req *models.AddAssigneeRequest) error {
	// First, verify the load exists
	load, err := s.loadRepo.GetByID(ctx, loadID)
	if err != nil {
		return fmt.Errorf("load not found: %w", err)
	}

	// Validate all assignees exist
	for _, a := range req.Assignees {
		exists, err := s.entityRepo.Exists(ctx, a.Email)
		if err != nil {
			return fmt.Errorf("failed to check assignee: %w", err)
		}
		if !exists {
			return fmt.Errorf("assignee not found: %s", a.Email)
		}
	}

	// Build assignments
	assignments := make([]models.LoadAssignment, 0, len(req.Assignees))
	for _, a := range req.Assignees {
		weight := a.Weight
		if weight == 0 {
			weight = 1.0 // Default weight
		}
		assignments = append(assignments, models.LoadAssignment{
			LoadID:      loadID,
			PersonEmail: a.Email,
			Weight:      weight,
		})
	}

	// Add assignments
	err = s.loadRepo.AddAssignees(ctx, loadID, assignments)
	if err != nil {
		return fmt.Errorf("failed to add assignees: %w", err)
	}

	// Trigger webhook alerts for affected persons (in background)
	for _, a := range req.Assignees {
		s.webhookService.CheckAndAlert(a.Email, load.Load.Date)
	}

	return nil
}

// RemoveAssignee removes a specific assignee from a load
func (s *LoadService) RemoveAssignee(ctx context.Context, loadID int, personEmail string) error {
	// Verify the load exists
	_, err := s.loadRepo.GetByID(ctx, loadID)
	if err != nil {
		return fmt.Errorf("load not found: %w", err)
	}

	// Remove the assignee
	err = s.loadRepo.RemoveAssignee(ctx, loadID, personEmail)
	if err != nil {
		return fmt.Errorf("failed to remove assignee: %w", err)
	}

	return nil
}
