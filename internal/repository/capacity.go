package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CapacityRepository struct {
	pool *pgxpool.Pool
}

func NewCapacityRepository(pool *pgxpool.Pool) *CapacityRepository {
	return &CapacityRepository{pool: pool}
}

// GetOverride retrieves a specific capacity override for an entity on a date
func (r *CapacityRepository) GetOverride(ctx context.Context, entityID string, date time.Time) (*models.CapacityOverride, error) {
	override := &models.CapacityOverride{}
	err := r.pool.QueryRow(ctx,
		`SELECT entity_id, date, capacity
		 FROM capacity_overrides WHERE entity_id = $1 AND date = $2`,
		entityID, date.Truncate(24*time.Hour)).Scan(
		&override.EntityID, &override.Date, &override.Capacity)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No override found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get capacity override: %w", err)
	}

	return override, nil
}

// GetOverridesRange retrieves all capacity overrides for an entity within a date range
func (r *CapacityRepository) GetOverridesRange(ctx context.Context, entityID string, start, end time.Time) ([]models.CapacityOverride, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT entity_id, date, capacity
		 FROM capacity_overrides
		 WHERE entity_id = $1 AND date BETWEEN $2 AND $3
		 ORDER BY date`,
		entityID, start.Truncate(24*time.Hour), end.Truncate(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get capacity overrides: %w", err)
	}
	defer rows.Close()

	var overrides []models.CapacityOverride
	for rows.Next() {
		var o models.CapacityOverride
		if err := rows.Scan(&o.EntityID, &o.Date, &o.Capacity); err != nil {
			return nil, fmt.Errorf("failed to scan override: %w", err)
		}
		overrides = append(overrides, o)
	}

	return overrides, nil
}

// SetOverride creates or updates a capacity override
func (r *CapacityRepository) SetOverride(ctx context.Context, override *models.CapacityOverride) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO capacity_overrides (entity_id, date, capacity)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (entity_id, date) DO UPDATE SET capacity = EXCLUDED.capacity`,
		override.EntityID, override.Date.Truncate(24*time.Hour), override.Capacity)

	if err != nil {
		return fmt.Errorf("failed to set capacity override: %w", err)
	}

	return nil
}

// DeleteOverride removes a capacity override
func (r *CapacityRepository) DeleteOverride(ctx context.Context, entityID string, date time.Time) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM capacity_overrides WHERE entity_id = $1 AND date = $2`,
		entityID, date.Truncate(24*time.Hour))

	if err != nil {
		return fmt.Errorf("failed to delete capacity override: %w", err)
	}

	return nil
}

// GetEffectiveCapacity returns the effective capacity for an entity on a date
// (override if exists, otherwise default)
func (r *CapacityRepository) GetEffectiveCapacity(ctx context.Context, entityID string, date time.Time) (float64, error) {
	// First try to get override
	var capacity float64
	err := r.pool.QueryRow(ctx,
		`SELECT capacity FROM capacity_overrides WHERE entity_id = $1 AND date = $2`,
		entityID, date.Truncate(24*time.Hour)).Scan(&capacity)

	if err == nil {
		return capacity, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to get override: %w", err)
	}

	// Fall back to default capacity
	err = r.pool.QueryRow(ctx,
		`SELECT default_capacity FROM entities WHERE id = $1`, entityID).Scan(&capacity)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrEntityNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get default capacity: %w", err)
	}

	return capacity, nil
}

// GetCapacitiesForRange returns a map of date -> capacity for an entity
func (r *CapacityRepository) GetCapacitiesForRange(ctx context.Context, entityID string, start, end time.Time) (map[time.Time]float64, error) {
	// Get default capacity first
	var defaultCapacity float64
	err := r.pool.QueryRow(ctx,
		`SELECT default_capacity FROM entities WHERE id = $1`, entityID).Scan(&defaultCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to get default capacity: %w", err)
	}

	// Initialize map with default capacity for all days (use UTC)
	capacities := make(map[time.Time]float64)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		normalizedDate := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		capacities[normalizedDate] = defaultCapacity
	}

	// Get overrides and apply them
	overrides, err := r.GetOverridesRange(ctx, entityID, start, end)
	if err != nil {
		return nil, err
	}

	for _, o := range overrides {
		normalizedDate := time.Date(o.Date.Year(), o.Date.Month(), o.Date.Day(), 0, 0, 0, 0, time.UTC)
		capacities[normalizedDate] = o.Capacity
	}

	return capacities, nil
}
