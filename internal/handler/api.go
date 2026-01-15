package handler

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gti/heatmap-internal/internal/models"
	"github.com/gti/heatmap-internal/internal/repository"
	"github.com/gti/heatmap-internal/internal/service"
	"github.com/labstack/echo/v4"
)

type APIHandler struct {
	loadService *service.LoadService
	entityRepo  *repository.EntityRepository
	groupRepo   *repository.GroupRepository
	validate    *validator.Validate
}

func NewAPIHandler(
	loadService *service.LoadService,
	entityRepo *repository.EntityRepository,
	groupRepo *repository.GroupRepository,
) *APIHandler {
	return &APIHandler{
		loadService: loadService,
		entityRepo:  entityRepo,
		groupRepo:   groupRepo,
		validate:    validator.New(),
	}
}

// UpsertLoad handles the n8n integration endpoint for creating/updating loads
func (h *APIHandler) UpsertLoad(c echo.Context) error {
	var req models.UpsertLoadRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	loadID, err := h.loadService.UpsertLoad(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"load_id": loadID,
	})
}

// ListEntities returns all entities
func (h *APIHandler) ListEntities(c echo.Context) error {
	entityType := c.QueryParam("type")

	var entities []models.Entity
	var err error

	switch entityType {
	case "person":
		entities, err = h.entityRepo.ListPersons(c.Request().Context())
	case "group":
		entities, err = h.entityRepo.ListGroups(c.Request().Context())
	default:
		entities, err = h.entityRepo.ListAll(c.Request().Context())
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, entities)
}

// GetEntity returns a single entity by ID
func (h *APIHandler) GetEntity(c echo.Context) error {
	id := c.Param("id")

	entity, err := h.entityRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == repository.ErrEntityNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "entity not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, entity)
}

// CreateEntity creates a new entity
func (h *APIHandler) CreateEntity(c echo.Context) error {
	var req models.CreateEntityRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	capacity := req.DefaultCapacity
	if capacity == 0 {
		capacity = 5.0 // Default
	}

	entity := &models.Entity{
		ID:              req.ID,
		Title:           req.Title,
		Type:            models.EntityType(req.Type),
		DefaultCapacity: capacity,
	}

	if err := h.entityRepo.Create(c.Request().Context(), entity); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, entity)
}

// DeleteEntity deletes an entity
func (h *APIHandler) DeleteEntity(c echo.Context) error {
	id := c.Param("id")

	if err := h.entityRepo.Delete(c.Request().Context(), id); err != nil {
		if err == repository.ErrEntityNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "entity not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"success": "entity deleted",
	})
}

// GetGroupMembers returns members of a group
func (h *APIHandler) GetGroupMembers(c echo.Context) error {
	groupID := c.Param("id")

	members, err := h.groupRepo.GetMembers(c.Request().Context(), groupID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"group_id": groupID,
		"members":  members,
	})
}

// AddGroupMember adds a member to a group
func (h *APIHandler) AddGroupMember(c echo.Context) error {
	groupID := c.Param("id")

	var req models.AddGroupMemberRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Verify group exists
	group, err := h.entityRepo.GetByID(c.Request().Context(), groupID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "group not found",
		})
	}
	if group.Type != models.EntityTypeGroup {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "entity is not a group",
		})
	}

	// Verify person exists
	person, err := h.entityRepo.GetByID(c.Request().Context(), req.PersonEmail)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "person not found",
		})
	}
	if person.Type != models.EntityTypePerson {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "entity is not a person",
		})
	}

	if err := h.groupRepo.AddMember(c.Request().Context(), groupID, req.PersonEmail); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"success": "member added",
	})
}

// RemoveGroupMember removes a member from a group
func (h *APIHandler) RemoveGroupMember(c echo.Context) error {
	groupID := c.Param("id")
	memberEmail := c.Param("member")

	if err := h.groupRepo.RemoveMember(c.Request().Context(), groupID, memberEmail); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"success": "member removed",
	})
}
