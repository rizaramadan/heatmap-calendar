package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepository struct {
	pool *pgxpool.Pool
}

func NewGroupRepository(pool *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{pool: pool}
}

// GetMembers returns all member emails for a group
func (r *GroupRepository) GetMembers(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT person_email FROM group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, email)
	}

	return members, nil
}

// AddMember adds a person to a group
func (r *GroupRepository) AddMember(ctx context.Context, groupID, personEmail string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, person_email)
		 VALUES ($1, $2)
		 ON CONFLICT (group_id, person_email) DO NOTHING`,
		groupID, personEmail)

	if err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	return nil
}

// RemoveMember removes a person from a group
func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, personEmail string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM group_members WHERE group_id = $1 AND person_email = $2`,
		groupID, personEmail)

	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	return nil
}

// GetGroupsForPerson returns all groups a person belongs to
func (r *GroupRepository) GetGroupsForPerson(ctx context.Context, personEmail string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT group_id FROM group_members WHERE person_email = $1`, personEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get person's groups: %w", err)
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var groupID string
		if err := rows.Scan(&groupID); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, groupID)
	}

	return groups, nil
}

// IsMember checks if a person is a member of a group
func (r *GroupRepository) IsMember(ctx context.Context, groupID, personEmail string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND person_email = $2)`,
		groupID, personEmail).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return exists, nil
}
