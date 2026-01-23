package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEntityNotFound = errors.New("entity not found")

type EntityRepository struct {
	pool *pgxpool.Pool
}

func NewEntityRepository(pool *pgxpool.Pool) *EntityRepository {
	return &EntityRepository{pool: pool}
}

// GetByID retrieves an entity by its ID
func (r *EntityRepository) GetByID(ctx context.Context, id string) (*models.Entity, error) {
	entity := &models.Entity{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, type, employee_id, default_capacity, created_at
		 FROM entities WHERE id = $1`, id).Scan(
		&entity.ID, &entity.Title, &entity.Type, &entity.EmployeeID, &entity.DefaultCapacity, &entity.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEntityNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	return entity, nil
}

// Create creates a new entity
func (r *EntityRepository) Create(ctx context.Context, entity *models.Entity) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO entities (id, title, type, employee_id, default_capacity)
		 VALUES ($1, $2, $3, $4, $5)`,
		entity.ID, entity.Title, entity.Type, entity.EmployeeID, entity.DefaultCapacity)

	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	return nil
}

// Update updates an existing entity
func (r *EntityRepository) Update(ctx context.Context, entity *models.Entity) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE entities SET title = $2, employee_id = $3, default_capacity = $4 WHERE id = $1`,
		entity.ID, entity.Title, entity.EmployeeID, entity.DefaultCapacity)

	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEntityNotFound
	}

	return nil
}

// UpdateDefaultCapacity updates only the default capacity of an entity
func (r *EntityRepository) UpdateDefaultCapacity(ctx context.Context, id string, capacity float64) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE entities SET default_capacity = $2 WHERE id = $1`,
		id, capacity)

	if err != nil {
		return fmt.Errorf("failed to update capacity: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEntityNotFound
	}

	return nil
}

// ListPersons returns all person entities
func (r *EntityRepository) ListPersons(ctx context.Context) ([]models.Entity, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, type, employee_id, default_capacity, created_at
		 FROM entities WHERE type = 'person' ORDER BY title`)
	if err != nil {
		return nil, fmt.Errorf("failed to list persons: %w", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Title, &e.Type, &e.EmployeeID, &e.DefaultCapacity, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// ListGroups returns all group entities
func (r *EntityRepository) ListGroups(ctx context.Context) ([]models.Entity, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, type, employee_id, default_capacity, created_at
		 FROM entities WHERE type = 'group' ORDER BY title`)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Title, &e.Type, &e.EmployeeID, &e.DefaultCapacity, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// ListAll returns all entities
func (r *EntityRepository) ListAll(ctx context.Context) ([]models.Entity, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, type, employee_id, default_capacity, created_at
		 FROM entities ORDER BY type, title`)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var e models.Entity
		if err := rows.Scan(&e.ID, &e.Title, &e.Type, &e.EmployeeID, &e.DefaultCapacity, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// Delete deletes an entity by ID
func (r *EntityRepository) Delete(ctx context.Context, id string) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM entities WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrEntityNotFound
	}

	return nil
}

// Exists checks if an entity exists
func (r *EntityRepository) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM entities WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check entity existence: %w", err)
	}
	return exists, nil
}
