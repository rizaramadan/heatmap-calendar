package handler

import (
	"errors"
	"net/http"
	"strings"

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
// @Summary Upsert a load
// @Description Create or update a load item with assignments (for n8n integration)
// @Tags Loads
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param load body models.UpsertLoadRequest true "Load data to upsert"
// @Success 200 {object} map[string]interface{} "Success with load ID"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/loads/upsert [post]
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

// UpsertLoadByEmployeeID handles the endpoint for creating/updating loads using employee_id
// @Summary Upsert a load by employee ID
// @Description Create or update a load item with assignments using employee_id instead of email
// @Tags Loads
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param load body models.UpsertLoadByEmployeeIDRequest true "Load data to upsert"
// @Success 200 {object} map[string]interface{} "Success with load ID"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Assignee not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/loads/upsert-by-employee-id [post]
func (h *APIHandler) UpsertLoadByEmployeeID(c echo.Context) error {
	var req models.UpsertLoadByEmployeeIDRequest
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

	loadID, err := h.loadService.UpsertLoadByEmployeeID(c.Request().Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		}
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
// @Summary List all entities
// @Description Returns all entities or filters by type (person/group)
// @Tags Entities
// @Produce json
// @Param type query string false "Filter by entity type (person or group)"
// @Success 200 {array} models.Entity "List of entities"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/entities [get]
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
// @Summary Get entity by ID
// @Description Returns a single entity by its ID
// @Tags Entities
// @Produce json
// @Param id path string true "Entity ID (email for persons, string ID for groups)"
// @Success 200 {object} models.Entity "Entity details"
// @Failure 404 {object} map[string]string "Entity not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/entities/{id} [get]
func (h *APIHandler) GetEntity(c echo.Context) error {
	id := c.Param("id")

	entity, err := h.entityRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEntityNotFound) {
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
// @Summary Create a new entity
// @Description Create a new person or group entity
// @Tags Entities
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param entity body models.CreateEntityRequest true "Entity to create"
// @Success 201 {object} models.Entity "Created entity"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/entities [post]
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
		EmployeeID:      req.EmployeeID,
		DefaultCapacity: capacity,
	}

	if err := h.entityRepo.Create(c.Request().Context(), entity); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, entity)
}

// UpdateEntity updates an existing entity
// @Summary Update an entity
// @Description Update an entity's title, employee_id, and/or default_capacity
// @Tags Entities
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Entity ID"
// @Param entity body models.UpdateEntityRequest true "Entity fields to update"
// @Success 200 {object} models.Entity "Updated entity"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Entity not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/entities/{id} [put]
func (h *APIHandler) UpdateEntity(c echo.Context) error {
	id := c.Param("id")

	var req models.UpdateEntityRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	// Get existing entity
	entity, err := h.entityRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEntityNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "entity not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Update fields if provided
	if req.Title != nil {
		entity.Title = *req.Title
	}
	if req.EmployeeID != nil {
		entity.EmployeeID = req.EmployeeID
	}
	if req.DefaultCapacity != nil {
		entity.DefaultCapacity = *req.DefaultCapacity
	}

	// Save updated entity
	if err := h.entityRepo.Update(c.Request().Context(), entity); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, entity)
}

// DeleteEntity deletes an entity
// @Summary Delete an entity
// @Description Delete an entity by its ID
// @Tags Entities
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Entity ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Entity not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/entities/{id} [delete]
func (h *APIHandler) DeleteEntity(c echo.Context) error {
	id := c.Param("id")

	if err := h.entityRepo.Delete(c.Request().Context(), id); err != nil {
		if errors.Is(err, repository.ErrEntityNotFound) {
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
// @Summary Get group members
// @Description Returns all members of a group
// @Tags Groups
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Group ID"
// @Success 200 {object} map[string]interface{} "Group members"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/groups/{id}/members [get]
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
// @Summary Add member to group
// @Description Add a person to a group
// @Tags Groups
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Group ID"
// @Param member body models.AddGroupMemberRequest true "Member to add"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Group or person not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/groups/{id}/members [post]
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
// @Summary Remove member from group
// @Description Remove a person from a group
// @Tags Groups
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Group ID"
// @Param member path string true "Member email to remove"
// @Success 200 {object} map[string]string "Success message"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/groups/{id}/members/{member} [delete]
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

// AddAssigneesToLoad adds one or more assignees to an existing load
// @Summary Add assignees to a load
// @Description Add one or more assignees to an existing load with optional weight
// @Tags Loads
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Load ID"
// @Param assignees body models.AddAssigneeRequest true "Assignees to add"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Load not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/loads/{id}/assignees [post]
func (h *APIHandler) AddAssigneesToLoad(c echo.Context) error {
	loadID := 0
	if err := echo.PathParamsBinder(c).Int("id", &loadID).BindError(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid load ID",
		})
	}

	var req models.AddAssigneeRequest
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

	if err := h.loadService.AddAssignees(c.Request().Context(), loadID, &req); err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "load not found") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "load not found",
			})
		}
		if strings.HasPrefix(errMsg, "assignee not found") {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": errMsg,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"success": "assignees added",
	})
}

// RemoveAssigneeFromLoad removes a specific assignee from a load
// @Summary Remove assignee from load
// @Description Remove a specific assignee from a load
// @Tags Loads
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Load ID"
// @Param email path string true "Assignee email to remove"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Load or assignee not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/loads/{id}/assignees/{email} [delete]
func (h *APIHandler) RemoveAssigneeFromLoad(c echo.Context) error {
	loadID := 0
	if err := echo.PathParamsBinder(c).Int("id", &loadID).BindError(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid load ID",
		})
	}

	email := c.Param("email")
	if email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "email parameter is required",
		})
	}

	if err := h.loadService.RemoveAssignee(c.Request().Context(), loadID, email); err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "load not found") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "load not found",
			})
		}
		if strings.Contains(errMsg, "assignee not found for this load") {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "assignee not found for this load",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"success": "assignee removed",
	})
}
