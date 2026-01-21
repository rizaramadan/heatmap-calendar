package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
)

type WebhookService struct {
	webhookURL   string
	loadRepo     *repository.LoadRepository
	capacityRepo *repository.CapacityRepository
	client       *http.Client
}

func NewWebhookService(
	webhookURL string,
	loadRepo *repository.LoadRepository,
	capacityRepo *repository.CapacityRepository,
) *WebhookService {
	return &WebhookService{
		webhookURL:   webhookURL,
		loadRepo:     loadRepo,
		capacityRepo: capacityRepo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckAndAlert checks if a person is overloaded on a future date and sends webhook alert
// This runs in a goroutine to avoid blocking the main request
func (s *WebhookService) CheckAndAlert(ctx context.Context, personEmail string, date time.Time) {
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Only alert for future dates
		if date.Before(time.Now().Truncate(24 * time.Hour)) {
			return
		}

		// Skip if no webhook URL configured
		if s.webhookURL == "" {
			return
		}

		// Get total load for the person on this date
		load, err := s.loadRepo.GetPersonLoadForDate(ctx, personEmail, date)
		if err != nil {
			log.Printf("Webhook: failed to get load for %s on %s: %v", personEmail, date.Format("2006-01-02"), err)
			return
		}

		// Get effective capacity
		capacity, err := s.capacityRepo.GetEffectiveCapacity(ctx, personEmail, date)
		if err != nil {
			log.Printf("Webhook: failed to get capacity for %s: %v", personEmail, err)
			return
		}

		// Check if overloaded
		if load <= capacity {
			return
		}

		// Send webhook alert
		payload := models.WebhookAlertPayload{
			PersonEmail: personEmail,
			Date:        date,
			Load:        load,
			Capacity:    capacity,
			Message:     fmt.Sprintf("%s is overloaded on %s (load: %.1f, capacity: %.1f)", personEmail, date.Format("2006-01-02"), load, capacity),
		}

		if err := s.sendWebhook(payload); err != nil {
			log.Printf("Webhook: failed to send alert: %v", err)
			return
		}

		log.Printf("Webhook: sent overload alert for %s on %s", personEmail, date.Format("2006-01-02"))
	}()
}

// sendWebhook sends a JSON payload to the configured webhook URL
func (s *WebhookService) sendWebhook(payload models.WebhookAlertPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// CheckAllAffectedPersons checks and alerts for all persons affected by a load
func (s *WebhookService) CheckAllAffectedPersons(ctx context.Context, loadID int, date time.Time) {
	persons, err := s.loadRepo.GetAffectedPersons(ctx, loadID)
	if err != nil {
		log.Printf("Webhook: failed to get affected persons for load %d: %v", loadID, err)
		return
	}

	for _, email := range persons {
		s.CheckAndAlert(ctx, email, date)
	}
}
