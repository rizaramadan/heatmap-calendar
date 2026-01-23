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

	// Ensure all assignees exist, create missing ones
	for _, a := range req.Assignees {
		exists, err := s.entityRepo.Exists(ctx, a.Email)
		if err != nil {
			return 0, fmt.Errorf("failed to check assignee: %w", err)
		}
		if !exists {
			// Auto-create missing person entity with default capacity
			newEntity := &models.Entity{
				ID:              a.Email,
				Title:           a.Email, // Use email as default title
				Type:            models.EntityTypePerson,
				DefaultCapacity: 5.0, // Default daily capacity
			}
			if err := s.entityRepo.Create(ctx, newEntity); err != nil {
				return 0, fmt.Errorf("failed to create assignee %s: %w", a.Email, err)
			}
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
		s.webhookService.CheckAndAlert(ctx, a.Email, date)
	}

	return loadID, nil
}

// UpsertLoadByEmployeeID creates or updates a load with its assignments using employee_id
func (s *LoadService) UpsertLoadByEmployeeID(ctx context.Context, req *models.UpsertLoadByEmployeeIDRequest) (int, error) {
	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return 0, fmt.Errorf("invalid date format: %w", err)
	}

	// Map employee_id to entity email (ID)
	type assigneeMapping struct {
		employeeID string
		email      string
		weight     float64
	}
	assigneeMappings := make([]assigneeMapping, 0, len(req.Assignees))

	// Look up each assignee by employee_id
	for _, a := range req.Assignees {
		entity, err := s.entityRepo.GetByEmployeeID(ctx, a.EmployeeID)
		if err != nil {
			return 0, fmt.Errorf("assignee with employee_id %s not found: %w", a.EmployeeID, err)
		}

		weight := a.Weight
		if weight == 0 {
			weight = 1.0 // Default weight
		}

		assigneeMappings = append(assigneeMappings, assigneeMapping{
			employeeID: a.EmployeeID,
			email:      entity.ID,
			weight:     weight,
		})
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

	assignments := make([]models.LoadAssignment, 0, len(assigneeMappings))
	for _, a := range assigneeMappings {
		assignments = append(assignments, models.LoadAssignment{
			PersonEmail: a.email,
			Weight:      a.weight,
		})
	}

	// Upsert the load
	loadID, err := s.loadRepo.UpsertByExternalID(ctx, load, assignments)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert load: %w", err)
	}

	// Trigger webhook alerts for affected persons (in background)
	for _, a := range assigneeMappings {
		s.webhookService.CheckAndAlert(ctx, a.email, date)
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

	// Ensure all assignees exist, create missing ones
	for _, a := range req.Assignees {
		exists, err := s.entityRepo.Exists(ctx, a.Email)
		if err != nil {
			return fmt.Errorf("failed to check assignee: %w", err)
		}
		if !exists {
			// Auto-create missing person entity with default capacity
			newEntity := &models.Entity{
				ID:              a.Email,
				Title:           a.Email, // Use email as default title
				Type:            models.EntityTypePerson,
				DefaultCapacity: 5.0, // Default daily capacity
			}
			if err := s.entityRepo.Create(ctx, newEntity); err != nil {
				return fmt.Errorf("failed to create assignee %s: %w", a.Email, err)
			}
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
		s.webhookService.CheckAndAlert(ctx, a.Email, load.Load.Date)
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
