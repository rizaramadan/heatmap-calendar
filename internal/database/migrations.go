package database

import (
	"context"
	"fmt"
	"log"
	"time"
)

// RunMigrations creates the database schema
func (db *DB) RunMigrations(ctx context.Context) error {
	log.Println("Running database migrations...")

	// Create tables
	schema := `
	-- Create entities table (persons and groups)
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		type TEXT CHECK (type IN ('person', 'group')),
		default_capacity FLOAT DEFAULT 5.0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Create group_members table
	CREATE TABLE IF NOT EXISTS group_members (
		group_id TEXT REFERENCES entities(id) ON DELETE CASCADE,
		person_email TEXT REFERENCES entities(id) ON DELETE CASCADE,
		PRIMARY KEY (group_id, person_email)
	);

	-- Create capacity_overrides table
	CREATE TABLE IF NOT EXISTS capacity_overrides (
		entity_id TEXT REFERENCES entities(id) ON DELETE CASCADE,
		date DATE NOT NULL,
		capacity FLOAT NOT NULL,
		PRIMARY KEY (entity_id, date)
	);

	-- Create loads table
	CREATE TABLE IF NOT EXISTS loads (
		id SERIAL PRIMARY KEY,
		external_id TEXT UNIQUE,
		title TEXT NOT NULL,
		source TEXT,
		date DATE NOT NULL
	);

	-- Create load_assignments table
	CREATE TABLE IF NOT EXISTS load_assignments (
		load_id INTEGER REFERENCES loads(id) ON DELETE CASCADE,
		person_email TEXT REFERENCES entities(id) ON DELETE CASCADE,
		weight FLOAT DEFAULT 1.0,
		PRIMARY KEY (load_id, person_email)
	);

	-- Create indexes for performance
	CREATE INDEX IF NOT EXISTS idx_loads_date ON loads(date);
	CREATE INDEX IF NOT EXISTS idx_loads_external_id ON loads(external_id);
	CREATE INDEX IF NOT EXISTS idx_load_assignments_person ON load_assignments(person_email);
	CREATE INDEX IF NOT EXISTS idx_capacity_overrides_date ON capacity_overrides(entity_id, date);

	-- Create OTP sessions table (in-memory alternative would be better for production)
	CREATE TABLE IF NOT EXISTS otp_records (
		email TEXT PRIMARY KEY,
		otp TEXT NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL
	);

	-- Create sessions table
	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		email TEXT NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_email ON sessions(email);
	`

	_, err := db.Pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// SeedData populates the database with sample data if empty
func (db *DB) SeedData(ctx context.Context) error {
	// Check if data already exists
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM entities").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check entity count: %w", err)
	}

	if count > 0 {
		log.Println("Database already has data, skipping seed")
		return nil
	}

	log.Println("Seeding database with sample data...")

	// Create sample persons
	persons := []struct {
		ID       string
		Title    string
		Capacity float64
	}{
		{"alice@example.com", "Alice Johnson", 5.0},
		{"bob@example.com", "Bob Smith", 6.0},
		{"charlie@example.com", "Charlie Brown", 4.0},
	}

	for _, p := range persons {
		_, err := db.Pool.Exec(ctx,
			"INSERT INTO entities (id, title, type, default_capacity) VALUES ($1, $2, 'person', $3)",
			p.ID, p.Title, p.Capacity)
		if err != nil {
			return fmt.Errorf("failed to create person %s: %w", p.ID, err)
		}
	}

	// Create sample groups
	groups := []struct {
		ID      string
		Title   string
		Members []string
	}{
		{"engineering", "Engineering Team", []string{"alice@example.com", "bob@example.com"}},
		{"design", "Design Team", []string{"charlie@example.com"}},
	}

	for _, g := range groups {
		_, err := db.Pool.Exec(ctx,
			"INSERT INTO entities (id, title, type, default_capacity) VALUES ($1, $2, 'group', 10.0)",
			g.ID, g.Title)
		if err != nil {
			return fmt.Errorf("failed to create group %s: %w", g.ID, err)
		}

		for _, member := range g.Members {
			_, err := db.Pool.Exec(ctx,
				"INSERT INTO group_members (group_id, person_email) VALUES ($1, $2)",
				g.ID, member)
			if err != nil {
				return fmt.Errorf("failed to add member %s to group %s: %w", member, g.ID, err)
			}
		}
	}

	// Create sample loads for the next 30 days
	today := time.Now().Truncate(24 * time.Hour)
	sampleLoads := []struct {
		Title    string
		DaysFrom int
		Assignee string
		Weight   float64
	}{
		{"Code Review Sprint", 1, "alice@example.com", 2.0},
		{"Database Migration", 2, "bob@example.com", 3.0},
		{"UI Redesign", 3, "charlie@example.com", 2.5},
		{"API Integration", 4, "alice@example.com", 1.5},
		{"Security Audit", 5, "bob@example.com", 2.0},
		{"Performance Testing", 6, "alice@example.com", 1.0},
		{"Documentation Update", 7, "charlie@example.com", 1.0},
		{"Feature Development", 8, "alice@example.com", 3.0},
		{"Bug Fixes", 9, "bob@example.com", 2.0},
		{"Client Meeting Prep", 10, "charlie@example.com", 1.5},
		// Overloaded days for demo
		{"Major Release", 5, "alice@example.com", 4.0},   // Total 6.0 for Alice on day 5 (overload!)
		{"Sprint Planning", 5, "bob@example.com", 5.0},   // Total 7.0 for Bob on day 5 (overload!)
		{"Design Review", 10, "charlie@example.com", 4.0}, // Total 5.5 for Charlie (overload!)
		{"Team Sync", 15, "alice@example.com", 2.0},
		{"Code Freeze", 20, "bob@example.com", 1.5},
		{"Deployment", 25, "alice@example.com", 2.5},
		{"Retrospective", 30, "charlie@example.com", 1.0},
	}

	for i, load := range sampleLoads {
		loadDate := today.AddDate(0, 0, load.DaysFrom)
		externalID := fmt.Sprintf("seed-%d", i+1)

		var loadID int
		err := db.Pool.QueryRow(ctx,
			"INSERT INTO loads (external_id, title, source, date) VALUES ($1, $2, 'seed', $3) RETURNING id",
			externalID, load.Title, loadDate).Scan(&loadID)
		if err != nil {
			return fmt.Errorf("failed to create load %s: %w", load.Title, err)
		}

		_, err = db.Pool.Exec(ctx,
			"INSERT INTO load_assignments (load_id, person_email, weight) VALUES ($1, $2, $3)",
			loadID, load.Assignee, load.Weight)
		if err != nil {
			return fmt.Errorf("failed to assign load %s: %w", load.Title, err)
		}
	}

	log.Println("Database seeding completed successfully")
	return nil
}
