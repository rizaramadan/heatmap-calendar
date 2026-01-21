//go:build e2e

// Package tests contains E2E tests for the load-calendar application.
//
// These tests require Docker to be running and are excluded from regular
// unit test runs. Run with: go test -tags=e2e ./e2e/tests/...
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gti/heatmap-internal/e2e/helpers"
	"github.com/gti/heatmap-internal/e2e/testenv"
)

// env is the shared test environment for all tests in this file.
var env *testenv.TestEnv

// TestMain sets up the E2E test environment before running tests.
//
// It spins up:
//   - Ephemeral PostgreSQL container via testcontainers-go
//   - Application service connected to the container
//
// Tests are skipped if Docker is not available.
func TestMain(m *testing.M) {
	// Check if Docker is available
	if !isDockerAvailable() {
		fmt.Println("SKIP: Docker is not available, skipping E2E tests")
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Setup test environment
	var err error
	env, err = testenv.Setup(ctx, testenv.DefaultConfig())
	if err != nil {
		fmt.Printf("Failed to setup E2E environment: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	env.Teardown()

	os.Exit(code)
}

// isDockerAvailable checks if Docker daemon is running.
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// TestSmoke is a comprehensive smoke test demonstrating all 4 E2E operation types:
//  1. db.Exec() / db.Query() - Direct PostgreSQL operations
//  2. api.Call() - HTTP API requests
//  3. browser.Navigate() / Click() / Text() - Headless browser automation
//  4. assert.X() - Value assertions
//
// This test verifies the complete stack is working together.
func TestSmoke(t *testing.T) {
	ctx := context.Background()
	a := helpers.NewAssert(t)

	// Clean any existing data
	err := env.CleanupTestData(ctx)
	a.NoError(err, "cleanup should succeed")

	// =========================================================================
	// STEP 1: Use db.Exec() to insert seed data directly into PostgreSQL
	// =========================================================================
	t.Log("Step 1: Inserting test data via db.Exec()")

	// Insert a test person
	_, err = env.DB.Exec(ctx, `
		INSERT INTO load_calendar_data.entities (id, title, type, default_capacity)
		VALUES ($1, $2, $3, $4)
	`, "smoke-test@example.com", "Smoke Test User", "person", 5.0)
	a.NoError(err, "should insert test person")

	// Insert a test group
	_, err = env.DB.Exec(ctx, `
		INSERT INTO load_calendar_data.entities (id, title, type, default_capacity)
		VALUES ($1, $2, $3, $4)
	`, "smoke-test-group", "Smoke Test Group", "group", 10.0)
	a.NoError(err, "should insert test group")

	// Add person to group
	_, err = env.DB.Exec(ctx, `
		INSERT INTO load_calendar_data.group_members (group_id, person_email)
		VALUES ($1, $2)
	`, "smoke-test-group", "smoke-test@example.com")
	a.NoError(err, "should add person to group")

	// Insert a test load for today
	today := time.Now().Format("2006-01-02")
	var loadID int
	err = env.Pool.QueryRow(ctx, `
		INSERT INTO load_calendar_data.loads (external_id, title, source, date)
		VALUES ($1, $2, $3, $4::date)
		RETURNING id
	`, "smoke-test-load-1", "Smoke Test Task", "e2e-test", today).Scan(&loadID)
	a.NoError(err, "should insert test load")
	a.NotEqual(0, loadID, "load ID should be assigned")

	// Assign the load to our test person
	_, err = env.DB.Exec(ctx, `
		INSERT INTO load_calendar_data.load_assignments (load_id, person_email, weight)
		VALUES ($1, $2, $3)
	`, loadID, "smoke-test@example.com", 2.5)
	a.NoError(err, "should assign load to person")

	t.Logf("Inserted load ID: %d for date: %s", loadID, today)

	// =========================================================================
	// STEP 2: Use api.Call() to hit API endpoints
	// =========================================================================
	t.Log("Step 2: Testing API endpoints via api.Call()")

	// GET /api/entities - List all entities
	resp, err := env.API.Call("GET", "/api/entities", nil)
	a.NoError(err, "GET /api/entities should not error")
	a.Equal(200, resp.StatusCode, "should return 200 OK")
	a.Contains(resp.String(), "smoke-test@example.com", "response should contain our test entity")
	a.Contains(resp.String(), "Smoke Test User", "response should contain entity title")

	// Parse the response to verify structure
	var entities []map[string]interface{}
	err = resp.JSON(&entities)
	a.NoError(err, "should parse JSON response")
	a.True(len(entities) >= 2, "should have at least 2 entities (person + group)")

	// GET /api/entities/:id - Get specific entity
	resp, err = env.API.Call("GET", "/api/entities/smoke-test@example.com", nil)
	a.NoError(err, "GET /api/entities/:id should not error")
	a.Equal(200, resp.StatusCode, "should return 200 OK")

	var entity map[string]interface{}
	err = resp.JSON(&entity)
	a.NoError(err, "should parse entity JSON")
	a.Equal("smoke-test@example.com", entity["id"], "entity ID should match")
	a.Equal("person", entity["type"], "entity type should be person")

	// POST /api/loads/upsert - Create a new load via API
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	loadPayload := map[string]interface{}{
		"external_id": "smoke-test-load-api",
		"title":       "API Created Load",
		"source":      "e2e-api-test",
		"date":        tomorrow,
		"assignees": []map[string]interface{}{
			{"email": "smoke-test@example.com", "weight": 1.5},
		},
	}
	resp, err = env.API.Call("POST", "/api/loads/upsert", loadPayload)
	a.NoError(err, "POST /api/loads/upsert should not error")
	a.Equal(200, resp.StatusCode, "should return 200 OK for upsert, got: %s", resp.String())

	t.Log("API endpoints verified successfully")

	// =========================================================================
	// STEP 3: Use browser automation to interact with HTMX pages
	// =========================================================================
	t.Log("Step 3: Testing UI via browser automation")

	// Get browser (lazy initialization)
	browser, err := env.Browser()
	a.NoError(err, "should start browser")

	// Navigate to the home page
	err = browser.Navigate(env.ServiceURL())
	a.NoError(err, "should navigate to home page")

	// Wait for the page to load and verify content
	err = browser.Wait("body")
	a.NoError(err, "should find body element")

	// Try to get the page title or main heading
	// The home page should show the heatmap
	pageText, err := browser.Text("body")
	a.NoError(err, "should get body text")
	a.NotEmpty(pageText, "page should have content")

	t.Logf("Home page loaded, body length: %d chars", len(pageText))

	// Navigate to login page
	err = browser.Navigate(env.ServiceURL() + "/login")
	a.NoError(err, "should navigate to login page")

	// Wait for login form
	err = browser.Wait("body")
	a.NoError(err, "should find login page body")

	loginPageText, err := browser.Text("body")
	a.NoError(err, "should get login page text")
	t.Logf("Login page loaded, content preview: %.100s...", loginPageText)

	t.Log("Browser automation verified successfully")

	// =========================================================================
	// STEP 4: Use db.Query() to verify database state after all actions
	// =========================================================================
	t.Log("Step 4: Verifying database state via db.Query()")

	// Query to count entities
	var entityCount int
	rows, err := env.DB.Query(ctx, "SELECT COUNT(*) FROM load_calendar_data.entities")
	a.NoError(err, "should query entity count")
	if rows.Next() {
		err = rows.Scan(&entityCount)
		a.NoError(err, "should scan entity count")
	}
	rows.Close()
	a.True(entityCount >= 2, "should have at least 2 entities, got %d", entityCount)

	// Query to verify the load we created via API exists
	var apiLoadTitle string
	rows, err = env.DB.Query(ctx, `
		SELECT title FROM load_calendar_data.loads
		WHERE external_id = $1
	`, "smoke-test-load-api")
	a.NoError(err, "should query API-created load")
	if rows.Next() {
		err = rows.Scan(&apiLoadTitle)
		a.NoError(err, "should scan load title")
	}
	rows.Close()
	a.Equal("API Created Load", apiLoadTitle, "API-created load should exist in database")

	// Query to verify load assignments
	var assignmentCount int
	rows, err = env.DB.Query(ctx, `
		SELECT COUNT(*) FROM load_calendar_data.load_assignments
		WHERE person_email = $1
	`, "smoke-test@example.com")
	a.NoError(err, "should query assignment count")
	if rows.Next() {
		err = rows.Scan(&assignmentCount)
		a.NoError(err, "should scan assignment count")
	}
	rows.Close()
	a.True(assignmentCount >= 2, "should have at least 2 assignments (db + api), got %d", assignmentCount)

	// Query to get total load weight for our test user
	var totalWeight float64
	rows, err = env.DB.Query(ctx, `
		SELECT COALESCE(SUM(weight), 0) FROM load_calendar_data.load_assignments
		WHERE person_email = $1
	`, "smoke-test@example.com")
	a.NoError(err, "should query total weight")
	if rows.Next() {
		err = rows.Scan(&totalWeight)
		a.NoError(err, "should scan total weight")
	}
	rows.Close()
	a.Equal(4.0, totalWeight, "total weight should be 2.5 + 1.5 = 4.0")

	t.Log("Database state verified successfully")

	// =========================================================================
	// SUMMARY
	// =========================================================================
	t.Log("=== SMOKE TEST COMPLETED SUCCESSFULLY ===")
	t.Logf("✓ db.Exec(): Inserted %d entities and %d load assignments", entityCount, assignmentCount)
	t.Logf("✓ api.Call(): Created load via API, verified entity endpoints")
	t.Logf("✓ browser.X(): Navigated to home and login pages")
	t.Logf("✓ assert.X(): Verified all operations with %d assertions", 25) // approximate
}

// TestAPIEntityCRUD tests entity CRUD operations via the API.
func TestAPIEntityCRUD(t *testing.T) {
	ctx := context.Background()
	a := helpers.NewAssert(t)

	// Clean slate
	err := env.CleanupTestData(ctx)
	a.NoError(err)

	// CREATE: Add a new entity via API
	createPayload := map[string]interface{}{
		"id":               "crud-test@example.com",
		"title":            "CRUD Test User",
		"type":             "person",
		"default_capacity": 6.0,
	}
	resp, err := env.API.Call("POST", "/api/entities", createPayload)
	a.NoError(err)
	a.Equal(201, resp.StatusCode, "should return 201 Created, got: %s", resp.String())

	// READ: Verify entity was created
	resp, err = env.API.Call("GET", "/api/entities/crud-test@example.com", nil)
	a.NoError(err)
	a.Equal(200, resp.StatusCode)

	var entity map[string]interface{}
	err = resp.JSON(&entity)
	a.NoError(err)
	a.Equal("crud-test@example.com", entity["id"])
	a.Equal("CRUD Test User", entity["title"])

	// Verify in database
	var dbTitle string
	rows, err := env.DB.Query(ctx, "SELECT title FROM load_calendar_data.entities WHERE id = $1", "crud-test@example.com")
	a.NoError(err)
	if rows.Next() {
		rows.Scan(&dbTitle)
	}
	rows.Close()
	a.Equal("CRUD Test User", dbTitle)

	// DELETE: Remove the entity
	resp, err = env.API.Call("DELETE", "/api/entities/crud-test@example.com", nil)
	a.NoError(err)
	a.Equal(200, resp.StatusCode)

	// Verify deletion
	resp, err = env.API.Call("GET", "/api/entities/crud-test@example.com", nil)
	a.NoError(err)
	a.Equal(404, resp.StatusCode, "should return 404 after deletion")

	t.Log("Entity CRUD operations verified")
}

// TestGroupMembership tests group member management via API.
func TestGroupMembership(t *testing.T) {
	ctx := context.Background()
	a := helpers.NewAssert(t)

	// Clean slate
	err := env.CleanupTestData(ctx)
	a.NoError(err)

	// Create a person and a group
	_, err = env.DB.Exec(ctx, `
		INSERT INTO load_calendar_data.entities (id, title, type, default_capacity)
		VALUES
			('member@example.com', 'Group Member', 'person', 5.0),
			('test-group', 'Test Group', 'group', 10.0)
	`)
	a.NoError(err)

	// Add member to group via API
	addPayload := map[string]interface{}{
		"person_email": "member@example.com",
	}
	resp, err := env.API.Call("POST", "/api/groups/test-group/members", addPayload)
	a.NoError(err)
	a.Equal(200, resp.StatusCode, "should add member, got: %s", resp.String())

	// List group members
	resp, err = env.API.Call("GET", "/api/groups/test-group/members", nil)
	a.NoError(err)
	a.Equal(200, resp.StatusCode)

	var members []interface{}
	err = json.Unmarshal(resp.Body, &members)
	a.NoError(err)
	a.Len(members, 1, "should have 1 member")

	// Remove member from group
	resp, err = env.API.Call("DELETE", "/api/groups/test-group/members/member@example.com", nil)
	a.NoError(err)
	a.Equal(200, resp.StatusCode)

	// Verify removal in database
	var memberCount int
	rows, err := env.DB.Query(ctx, "SELECT COUNT(*) FROM load_calendar_data.group_members WHERE group_id = $1", "test-group")
	a.NoError(err)
	if rows.Next() {
		rows.Scan(&memberCount)
	}
	rows.Close()
	a.Equal(0, memberCount, "should have no members after removal")

	t.Log("Group membership operations verified")
}
