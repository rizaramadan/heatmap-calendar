// @title Heatmap Internal API
// @version 1.0
// @description API for managing heatmap loads, entities, groups, and capacity tracking
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name x-api-key
// @description API Key for protected endpoints

package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/gti/heatmap-internal/docs"
	"github.com/gti/heatmap-internal/internal/config"
	"github.com/gti/heatmap-internal/internal/database"
	"github.com/gti/heatmap-internal/internal/handler"
	"github.com/gti/heatmap-internal/internal/middleware"
	"github.com/gti/heatmap-internal/internal/repository"
	"github.com/gti/heatmap-internal/internal/service"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	ctx := context.Background()
	if err := db.RunMigrations(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed data
	if err := db.SeedData(ctx); err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}

	// Initialize repositories
	entityRepo := repository.NewEntityRepository(db.Pool)
	groupRepo := repository.NewGroupRepository(db.Pool)
	capacityRepo := repository.NewCapacityRepository(db.Pool)
	loadRepo := repository.NewLoadRepository(db.Pool)

	// Initialize services
	webhookService := service.NewWebhookService(cfg.WebhookDestinationURL, loadRepo, capacityRepo)
	heatmapService := service.NewHeatmapService(entityRepo, capacityRepo, loadRepo, groupRepo)
	loadService := service.NewLoadService(loadRepo, entityRepo, webhookService)
	authService := service.NewAuthService(db.Pool, cfg.LarkBearerToken)
	capacityService := service.NewCapacityService(entityRepo, capacityRepo)

	// Load templates
	templates, err := loadTemplates()
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Initialize handlers
	heatmapHandler := handler.NewHeatmapHandler(heatmapService, entityRepo, templates)
	apiHandler := handler.NewAPIHandler(loadService, entityRepo, groupRepo)
	authHandler := handler.NewAuthHandler(authService, entityRepo, templates)
	capacityHandler := handler.NewCapacityHandler(capacityService, templates)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORS())

	// Optional session auth for all routes (sets user context if logged in)
	e.Use(middleware.SessionAuthOptional(authService))

	// Public routes
	e.GET("/", heatmapHandler.Index)
	e.GET("/login", authHandler.LoginPage)

	// Auth routes (public)
	e.POST("/auth/request-otp", authHandler.RequestOTP)
	e.POST("/auth/verify-otp", authHandler.VerifyOTP)
	e.POST("/auth/logout", authHandler.Logout)

	// Protected routes (require session)
	protected := e.Group("")
	protected.Use(middleware.SessionAuth(authService))
	protected.GET("/my-capacity", capacityHandler.MyCapacityPage)
	protected.POST("/api/my-capacity", capacityHandler.UpdateMyCapacity)
	protected.DELETE("/api/my-capacity/override/:date", capacityHandler.DeleteMyCapacityOverride)

	// Public API routes
	e.GET("/api/entities", apiHandler.ListEntities)
	e.GET("/api/entities/:id", apiHandler.GetEntity)
	e.GET("/api/heatmap/:entity", heatmapHandler.GetHeatmapPartial)
	e.GET("/api/heatmap/:entity/day/:date", heatmapHandler.GetDayDetails)

	// Protected API routes (require x-api-key)
	apiProtected := e.Group("/api")
	apiProtected.Use(middleware.APIKeyAuth(cfg.APIKey))
	apiProtected.POST("/loads/upsert", apiHandler.UpsertLoad)
	apiProtected.POST("/loads/:id/assignees", apiHandler.AddAssigneesToLoad)
	apiProtected.DELETE("/loads/:id/assignees/:email", apiHandler.RemoveAssigneeFromLoad)
	apiProtected.POST("/entities", apiHandler.CreateEntity)
	apiProtected.DELETE("/entities/:id", apiHandler.DeleteEntity)
	apiProtected.GET("/groups/:id/members", apiHandler.GetGroupMembers)
	apiProtected.POST("/groups/:id/members", apiHandler.AddGroupMember)
	apiProtected.DELETE("/groups/:id/members/:member", apiHandler.RemoveGroupMember)

	// Static files (if needed)
	e.Static("/static", "static")

	// Swagger API documentation
	e.GET("/api/doc/*", echoSwagger.WrapHandler)

	// Start server in goroutine
	go func() {
		addr := ":" + cfg.Port
		log.Printf("Starting server on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func loadTemplates() (*template.Template, error) {
	// Custom template functions
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 3:04 PM")
		},
	}

	templates, err := template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}

	templates, err = templates.ParseGlob("templates/partials/*.html")
	if err != nil {
		return nil, err
	}

	return templates, nil
}
