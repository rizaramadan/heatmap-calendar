package models

import (
	"time"
)

// EntityType represents the type of entity (person or group)
type EntityType string

const (
	EntityTypePerson EntityType = "person"
	EntityTypeGroup  EntityType = "group"
)

// Entity represents a person or group in the system
type Entity struct {
	ID              string     `json:"id"`               // email for persons, string-id for groups
	Title           string     `json:"title"`            // Display name
	Type            EntityType `json:"type"`             // "person" or "group"
	EmployeeID      *string    `json:"employee_id,omitempty"` // Optional employee identifier
	DefaultCapacity float64    `json:"default_capacity"` // Default daily capacity
	CreatedAt       time.Time  `json:"created_at"`
}

// GroupMember represents the relationship between a group and its members
type GroupMember struct {
	GroupID     string `json:"group_id"`
	PersonEmail string `json:"person_email"`
}

// CapacityOverride represents a date-specific capacity override for an entity
type CapacityOverride struct {
	EntityID string    `json:"entity_id"`
	Date     time.Time `json:"date"`
	Capacity float64   `json:"capacity"`
}

// Load represents a task/load item
type Load struct {
	ID         int       `json:"id"`
	ExternalID *string   `json:"external_id,omitempty"` // For n8n/external system deduplication
	Title      string    `json:"title"`
	Source     *string   `json:"source,omitempty"` // Origin system (gcal, crm, etc.)
	URL        *string   `json:"url,omitempty"`    // Link back to original platform (gcal, lark, etc.)
	Date       time.Time `json:"date"`
}

// LoadAssignment represents the assignment of a load to a person with a weight
type LoadAssignment struct {
	LoadID      int     `json:"load_id"`
	PersonEmail string  `json:"person_email"`
	Weight      float64 `json:"weight"` // Default 1.0
}

// LoadWithAssignments combines a load with its assignments
type LoadWithAssignments struct {
	Load        Load             `json:"load"`
	Assignments []LoadAssignment `json:"assignments"`
}

// HeatmapDay represents a single day in the heatmap
type HeatmapDay struct {
	Date     time.Time `json:"date"`
	Load     float64   `json:"load"`
	Capacity float64   `json:"capacity"`
	Color    string    `json:"color"`
}

// HeatmapData represents the complete heatmap data for an entity
type HeatmapData struct {
	Entity Entity       `json:"entity"`
	Days   []HeatmapDay `json:"days"`
}

// OTPRecord stores OTP information for authentication
type OTPRecord struct {
	Email     string
	OTP       string
	ExpiresAt time.Time
}

// Session represents a user session
type Session struct {
	Token     string
	Email     string
	ExpiresAt time.Time
}

// --- API Request/Response Types ---

// UpsertLoadRequest is the request body for the n8n load upsert endpoint
type UpsertLoadRequest struct {
	ExternalID string `json:"external_id" validate:"required"`
	Title      string `json:"title" validate:"required"`
	Source     string `json:"source,omitempty"`
	URL        string `json:"url,omitempty"` // Link back to original platform
	Date       string `json:"date" validate:"required"` // Format: YYYY-MM-DD
	Assignees  []struct {
		Email  string  `json:"email" validate:"required,email"`
		Weight float64 `json:"weight,omitempty"` // Default 1.0
	} `json:"assignees" validate:"required,min=1,dive"`
}

// UpsertLoadByEmployeeIDRequest is the request body for the load upsert endpoint using employee_id
type UpsertLoadByEmployeeIDRequest struct {
	ExternalID string `json:"external_id" validate:"required"`
	Title      string `json:"title" validate:"required"`
	Source     string `json:"source,omitempty"`
	URL        string `json:"url,omitempty"` // Link back to original platform
	Date       string `json:"date" validate:"required"` // Format: YYYY-MM-DD
	Assignees  []struct {
		EmployeeID string  `json:"employee_id" validate:"required"`
		Weight     float64 `json:"weight,omitempty"` // Default 1.0
	} `json:"assignees" validate:"required,min=1,dive"`
}

// CreateEntityRequest is the request body for creating an entity
type CreateEntityRequest struct {
	ID              string  `json:"id" validate:"required"`
	Title           string  `json:"title" validate:"required"`
	Type            string  `json:"type" validate:"required,oneof=person group"`
	EmployeeID      *string `json:"employee_id,omitempty"`
	DefaultCapacity float64 `json:"default_capacity,omitempty"`
}

// UpdateEntityRequest is the request body for updating an entity
type UpdateEntityRequest struct {
	Title           *string  `json:"title,omitempty"`
	EmployeeID      *string  `json:"employee_id,omitempty"`
	DefaultCapacity *float64 `json:"default_capacity,omitempty"`
}

// UpdateCapacityRequest is the request body for updating capacity
type UpdateCapacityRequest struct {
	DefaultCapacity *float64 `json:"default_capacity,omitempty"`
	DateOverrides   []struct {
		Date     string  `json:"date" validate:"required"` // Format: YYYY-MM-DD
		Capacity float64 `json:"capacity" validate:"required,min=0"`
	} `json:"date_overrides,omitempty"`
}

// OTPRequest is the request body for requesting an OTP
type OTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// VerifyOTPRequest is the request body for verifying an OTP
type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

// WebhookAlertPayload is sent to the webhook destination when overload is detected
type WebhookAlertPayload struct {
	PersonEmail string    `json:"person_email"`
	Date        time.Time `json:"date"`
	Load        float64   `json:"load"`
	Capacity    float64   `json:"capacity"`
	Message     string    `json:"message"`
}

// AddGroupMemberRequest is the request body for adding a member to a group
type AddGroupMemberRequest struct {
	PersonEmail string `json:"person_email" validate:"required,email"`
}

// AddAssigneeRequest is the request body for adding assignee(s) to a load
type AddAssigneeRequest struct {
	Assignees []struct {
		Email  string  `json:"email" validate:"required,email"`
		Weight float64 `json:"weight,omitempty"` // Default 1.0
	} `json:"assignees" validate:"required,min=1,dive"`
}
