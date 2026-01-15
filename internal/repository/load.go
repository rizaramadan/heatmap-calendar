package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gti/heatmap-internal/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoadRepository struct {
	pool *pgxpool.Pool
}

func NewLoadRepository(pool *pgxpool.Pool) *LoadRepository {
	return &LoadRepository{pool: pool}
}

// UpsertByExternalID creates or updates a load and its assignments by external ID
func (r *LoadRepository) UpsertByExternalID(ctx context.Context, load *models.Load, assignments []models.LoadAssignment) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var loadID int

	// Upsert the load
	err = tx.QueryRow(ctx,
		`INSERT INTO loads (external_id, title, source, date)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (external_id) DO UPDATE SET
		   title = EXCLUDED.title,
		   source = EXCLUDED.source,
		   date = EXCLUDED.date
		 RETURNING id`,
		load.ExternalID, load.Title, load.Source, load.Date.Truncate(24*time.Hour)).Scan(&loadID)

	if err != nil {
		return 0, fmt.Errorf("failed to upsert load: %w", err)
	}

	// Delete existing assignments for this load
	_, err = tx.Exec(ctx, `DELETE FROM load_assignments WHERE load_id = $1`, loadID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old assignments: %w", err)
	}

	// Insert new assignments
	for _, a := range assignments {
		_, err = tx.Exec(ctx,
			`INSERT INTO load_assignments (load_id, person_email, weight)
			 VALUES ($1, $2, $3)`,
			loadID, a.PersonEmail, a.Weight)
		if err != nil {
			return 0, fmt.Errorf("failed to insert assignment: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return loadID, nil
}

// GetByID retrieves a load by its ID
func (r *LoadRepository) GetByID(ctx context.Context, id int) (*models.LoadWithAssignments, error) {
	load := &models.Load{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, external_id, title, source, date FROM loads WHERE id = $1`, id).Scan(
		&load.ID, &load.ExternalID, &load.Title, &load.Source, &load.Date)
	if err != nil {
		return nil, fmt.Errorf("failed to get load: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT load_id, person_email, weight FROM load_assignments WHERE load_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get assignments: %w", err)
	}
	defer rows.Close()

	var assignments []models.LoadAssignment
	for rows.Next() {
		var a models.LoadAssignment
		if err := rows.Scan(&a.LoadID, &a.PersonEmail, &a.Weight); err != nil {
			return nil, fmt.Errorf("failed to scan assignment: %w", err)
		}
		assignments = append(assignments, a)
	}

	return &models.LoadWithAssignments{
		Load:        *load,
		Assignments: assignments,
	}, nil
}

// GetLoadsByDateRange retrieves all loads within a date range
func (r *LoadRepository) GetLoadsByDateRange(ctx context.Context, start, end time.Time) ([]models.LoadWithAssignments, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT l.id, l.external_id, l.title, l.source, l.date,
		        la.person_email, la.weight
		 FROM loads l
		 LEFT JOIN load_assignments la ON l.id = la.load_id
		 WHERE l.date BETWEEN $1 AND $2
		 ORDER BY l.date, l.id`,
		start.Truncate(24*time.Hour), end.Truncate(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get loads: %w", err)
	}
	defer rows.Close()

	loadMap := make(map[int]*models.LoadWithAssignments)
	var loadOrder []int

	for rows.Next() {
		var (
			loadID      int
			externalID  *string
			title       string
			source      *string
			date        time.Time
			personEmail *string
			weight      *float64
		)

		if err := rows.Scan(&loadID, &externalID, &title, &source, &date, &personEmail, &weight); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if _, exists := loadMap[loadID]; !exists {
			loadMap[loadID] = &models.LoadWithAssignments{
				Load: models.Load{
					ID:         loadID,
					ExternalID: externalID,
					Title:      title,
					Source:     source,
					Date:       date,
				},
				Assignments: []models.LoadAssignment{},
			}
			loadOrder = append(loadOrder, loadID)
		}

		if personEmail != nil {
			loadMap[loadID].Assignments = append(loadMap[loadID].Assignments, models.LoadAssignment{
				LoadID:      loadID,
				PersonEmail: *personEmail,
				Weight:      *weight,
			})
		}
	}

	result := make([]models.LoadWithAssignments, 0, len(loadOrder))
	for _, id := range loadOrder {
		result = append(result, *loadMap[id])
	}

	return result, nil
}

// GetPersonLoadForDateRange returns the total load per day for a person
func (r *LoadRepository) GetPersonLoadForDateRange(ctx context.Context, email string, start, end time.Time) (map[time.Time]float64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT l.date, COALESCE(SUM(la.weight), 0) as total_load
		 FROM loads l
		 JOIN load_assignments la ON l.id = la.load_id
		 WHERE la.person_email = $1 AND l.date BETWEEN $2 AND $3
		 GROUP BY l.date`,
		email, start.Truncate(24*time.Hour), end.Truncate(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get person load: %w", err)
	}
	defer rows.Close()

	loads := make(map[time.Time]float64)
	for rows.Next() {
		var date time.Time
		var load float64
		if err := rows.Scan(&date, &load); err != nil {
			return nil, fmt.Errorf("failed to scan load: %w", err)
		}
		// Use UTC to normalize the date key
		normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		loads[normalizedDate] = load
	}

	return loads, nil
}

// GetGroupLoadForDateRange returns the total load per day for a group (sum of all members)
// This is the "killer query" from the spec
func (r *LoadRepository) GetGroupLoadForDateRange(ctx context.Context, groupID string, start, end time.Time) (map[time.Time]float64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
			l.date,
			COALESCE(SUM(la.weight), 0) as total_load
		 FROM loads l
		 JOIN load_assignments la ON l.id = la.load_id
		 JOIN group_members gm ON la.person_email = gm.person_email
		 WHERE gm.group_id = $1 AND l.date BETWEEN $2 AND $3
		 GROUP BY l.date`,
		groupID, start.Truncate(24*time.Hour), end.Truncate(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get group load: %w", err)
	}
	defer rows.Close()

	loads := make(map[time.Time]float64)
	for rows.Next() {
		var date time.Time
		var load float64
		if err := rows.Scan(&date, &load); err != nil {
			return nil, fmt.Errorf("failed to scan load: %w", err)
		}
		// Use UTC to normalize the date key
		normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		loads[normalizedDate] = load
	}

	return loads, nil
}

// GetPersonLoadForDate returns the total load for a person on a specific date
func (r *LoadRepository) GetPersonLoadForDate(ctx context.Context, email string, date time.Time) (float64, error) {
	var load float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(la.weight), 0) as total_load
		 FROM loads l
		 JOIN load_assignments la ON l.id = la.load_id
		 WHERE la.person_email = $1 AND l.date = $2`,
		email, date.Truncate(24*time.Hour)).Scan(&load)
	if err != nil {
		return 0, fmt.Errorf("failed to get person load: %w", err)
	}
	return load, nil
}

// GetLoadsForEntityOnDate returns all loads for an entity (person or group members) on a specific date
func (r *LoadRepository) GetLoadsForEntityOnDate(ctx context.Context, entityID string, entityType models.EntityType, date time.Time) ([]models.LoadWithAssignments, error) {
	var query string
	if entityType == models.EntityTypePerson {
		query = `
			SELECT l.id, l.external_id, l.title, l.source, l.date,
			       la.person_email, la.weight
			FROM loads l
			JOIN load_assignments la ON l.id = la.load_id
			WHERE la.person_email = $1 AND l.date = $2
			ORDER BY l.id`
	} else {
		query = `
			SELECT DISTINCT l.id, l.external_id, l.title, l.source, l.date,
			       la.person_email, la.weight
			FROM loads l
			JOIN load_assignments la ON l.id = la.load_id
			JOIN group_members gm ON la.person_email = gm.person_email
			WHERE gm.group_id = $1 AND l.date = $2
			ORDER BY l.id`
	}

	rows, err := r.pool.Query(ctx, query, entityID, date.Truncate(24*time.Hour))
	if err != nil {
		return nil, fmt.Errorf("failed to get loads: %w", err)
	}
	defer rows.Close()

	loadMap := make(map[int]*models.LoadWithAssignments)
	var loadOrder []int

	for rows.Next() {
		var (
			loadID      int
			externalID  *string
			title       string
			source      *string
			loadDate    time.Time
			personEmail string
			weight      float64
		)

		if err := rows.Scan(&loadID, &externalID, &title, &source, &loadDate, &personEmail, &weight); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if _, exists := loadMap[loadID]; !exists {
			loadMap[loadID] = &models.LoadWithAssignments{
				Load: models.Load{
					ID:         loadID,
					ExternalID: externalID,
					Title:      title,
					Source:     source,
					Date:       loadDate,
				},
				Assignments: []models.LoadAssignment{},
			}
			loadOrder = append(loadOrder, loadID)
		}

		loadMap[loadID].Assignments = append(loadMap[loadID].Assignments, models.LoadAssignment{
			LoadID:      loadID,
			PersonEmail: personEmail,
			Weight:      weight,
		})
	}

	result := make([]models.LoadWithAssignments, 0, len(loadOrder))
	for _, id := range loadOrder {
		result = append(result, *loadMap[id])
	}

	return result, nil
}

// GetAffectedPersons returns all persons assigned to a load
func (r *LoadRepository) GetAffectedPersons(ctx context.Context, loadID int) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT person_email FROM load_assignments WHERE load_id = $1`, loadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get affected persons: %w", err)
	}
	defer rows.Close()

	var persons []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("failed to scan person: %w", err)
		}
		persons = append(persons, email)
	}

	return persons, nil
}

// Delete deletes a load by ID
func (r *LoadRepository) Delete(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM loads WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete load: %w", err)
	}
	return nil
}
